package jspacker

import (
	"fmt"
	"sort"
	"strconv"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/parser"
)

func jToken(data string) *javascript.Token {
	return &javascript.Token{Token: parser.Token{Data: data}}
}

func (c *config) makeLoader() error {
	promise := &javascript.MemberExpression{
		PrimaryExpression: &javascript.PrimaryExpression{
			IdentifierReference: jToken("Promise"),
		},
	}
	promiseResolve := &javascript.MemberExpression{
		MemberExpression: promise,
		IdentifierName:   jToken("resolve"),
	}
	trueAE := &javascript.AssignmentExpression{
		ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
			Literal: jToken("true"),
		}),
	}
	object := &javascript.MemberExpression{
		PrimaryExpression: &javascript.PrimaryExpression{
			IdentifierReference: jToken("Object"),
		},
	}
	url := jToken("url")
	wrappedURL := javascript.WrapConditional(&javascript.PrimaryExpression{
		IdentifierReference: url,
	})
	importURL := javascript.WrapConditional(&javascript.CallExpression{
		ImportCall: &javascript.AssignmentExpression{
			ConditionalExpression: wrappedURL,
		},
	})

	var include *javascript.AssignmentExpression

	if !c.bare || c.parseDynamic {
		exportArr := &javascript.ArrayLiteral{
			ElementList: make([]javascript.ArrayElement, 0, len(c.filesDone)),
		}

		urls := make([]string, 0, len(c.filesDone))
		imports := jToken("imports")
		importsGet := &javascript.MemberExpression{
			MemberExpression: &javascript.MemberExpression{
				PrimaryExpression: &javascript.PrimaryExpression{
					IdentifierReference: imports,
				},
			},
			IdentifierName: jToken("get"),
		}

		for url := range c.filesDone {
			urls = append(urls, url)
		}

		sort.Strings(urls)

		for _, url := range urls {
			d := c.filesDone[url]

			if c.bare && !d.dynamicRequirement {
				continue
			}

			el := append(make([]javascript.ArrayElement, 0, len(d.exports)+1), javascript.ArrayElement{
				AssignmentExpression: javascript.AssignmentExpression{
					ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
						Literal: jToken(strconv.Quote(url)),
					}),
				},
			})

			props := make([]string, 0, len(d.exports))

			for prop := range d.exports {
				props = append(props, prop)
			}

			sort.Strings(props)

			for _, prop := range props {
				binding := d.exports[prop]
				propName := javascript.ArrayElement{
					AssignmentExpression: javascript.AssignmentExpression{
						ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
							Literal: jToken(strconv.Quote(prop)),
						}),
					},
				}

				var ael []javascript.ArrayElement

				if binding.binding == "" {
					ael = []javascript.ArrayElement{
						propName,
						{
							AssignmentExpression: javascript.AssignmentExpression{
								ArrowFunction: &javascript.ArrowFunction{
									FormalParameters: &javascript.FormalParameters{},
									AssignmentExpression: &javascript.AssignmentExpression{
										ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
											MemberExpression: importsGet,
											Arguments: &javascript.Arguments{
												ArgumentList: []javascript.Argument{
													{
														AssignmentExpression: javascript.AssignmentExpression{
															ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
																Literal: jToken(strconv.Quote(binding.dependency.url)),
															}),
														},
													},
												},
											},
										}),
									},
								},
							},
						},
					}
				} else {
					b := d.resolveExport(prop)
					if b == nil {
						return fmt.Errorf("error resolving export %s (%s): %w", prop, d.url, ErrInvalidExport)
					}

					ael = []javascript.ArrayElement{
						propName,
						{
							AssignmentExpression: javascript.AssignmentExpression{
								ArrowFunction: &javascript.ArrowFunction{
									FormalParameters: &javascript.FormalParameters{},
									AssignmentExpression: &javascript.AssignmentExpression{
										ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
											IdentifierReference: b.Token,
										}),
									},
								},
							},
						},
					}
				}

				el = append(el, javascript.ArrayElement{
					AssignmentExpression: javascript.AssignmentExpression{
						ConditionalExpression: javascript.WrapConditional(&javascript.ArrayLiteral{
							ElementList: ael,
						}),
					},
				})
			}

			exportArr.ElementList = append(exportArr.ElementList, javascript.ArrayElement{
				AssignmentExpression: javascript.AssignmentExpression{
					ConditionalExpression: javascript.WrapConditional(&javascript.ArrayLiteral{
						ElementList: el,
					}),
				},
			})
		}

		if len(exportArr.ElementList) > 0 {
			mapt := jToken("map")
			prop := jToken("prop")
			get := jToken("get")
			props := jToken("props")
			include = &javascript.AssignmentExpression{
				ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
					MemberExpression: &javascript.MemberExpression{
						PrimaryExpression: &javascript.PrimaryExpression{
							ParenthesizedExpression: &javascript.ParenthesizedExpression{
								Expressions: []javascript.AssignmentExpression{
									{
										ArrowFunction: &javascript.ArrowFunction{
											FormalParameters: &javascript.FormalParameters{},
											FunctionBody: &javascript.Block{
												StatementList: []javascript.StatementListItem{
													{
														Declaration: &javascript.Declaration{
															LexicalDeclaration: &javascript.LexicalDeclaration{
																LetOrConst: javascript.Const,
																BindingList: []javascript.LexicalBinding{
																	{
																		BindingIdentifier: imports,
																		Initializer: &javascript.AssignmentExpression{
																			ConditionalExpression: javascript.WrapConditional(javascript.MemberExpression{
																				MemberExpression: &javascript.MemberExpression{
																					PrimaryExpression: &javascript.PrimaryExpression{
																						IdentifierReference: jToken("Map"),
																					},
																				},
																				Arguments: &javascript.Arguments{
																					ArgumentList: []javascript.Argument{
																						{
																							AssignmentExpression: javascript.AssignmentExpression{
																								ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																									MemberExpression: &javascript.MemberExpression{
																										MemberExpression: &javascript.MemberExpression{
																											PrimaryExpression: &javascript.PrimaryExpression{
																												ArrayLiteral: exportArr,
																											},
																										},
																										IdentifierName: mapt,
																									},
																									Arguments: &javascript.Arguments{
																										ArgumentList: []javascript.Argument{
																											{
																												AssignmentExpression: javascript.AssignmentExpression{
																													ArrowFunction: &javascript.ArrowFunction{
																														FormalParameters: &javascript.FormalParameters{
																															FormalParameterList: []javascript.BindingElement{
																																{
																																	ArrayBindingPattern: &javascript.ArrayBindingPattern{
																																		BindingElementList: []javascript.BindingElement{
																																			{
																																				SingleNameBinding: url,
																																			},
																																		},
																																		BindingRestElement: &javascript.BindingElement{
																																			SingleNameBinding: props,
																																		},
																																	},
																																},
																															},
																														},
																														AssignmentExpression: &javascript.AssignmentExpression{
																															ConditionalExpression: javascript.WrapConditional(&javascript.ArrayLiteral{
																																ElementList: []javascript.ArrayElement{
																																	{
																																		AssignmentExpression: javascript.AssignmentExpression{
																																			ConditionalExpression: wrappedURL,
																																		},
																																	},
																																	{
																																		AssignmentExpression: javascript.AssignmentExpression{
																																			ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																																				MemberExpression: &javascript.MemberExpression{
																																					MemberExpression: object,
																																					IdentifierName:   jToken("freeze"),
																																				},
																																				Arguments: &javascript.Arguments{
																																					ArgumentList: []javascript.Argument{
																																						{
																																							AssignmentExpression: javascript.AssignmentExpression{
																																								ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																																									MemberExpression: &javascript.MemberExpression{
																																										MemberExpression: object,
																																										IdentifierName:   jToken("defineProperties"),
																																									},
																																									Arguments: &javascript.Arguments{
																																										ArgumentList: []javascript.Argument{
																																											{
																																												AssignmentExpression: javascript.AssignmentExpression{
																																													ConditionalExpression: javascript.WrapConditional(&javascript.ObjectLiteral{}),
																																												},
																																											},
																																											{
																																												AssignmentExpression: javascript.AssignmentExpression{
																																													ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																																														MemberExpression: &javascript.MemberExpression{
																																															MemberExpression: object,
																																															IdentifierName:   jToken("fromEntries"),
																																														},
																																														Arguments: &javascript.Arguments{
																																															ArgumentList: []javascript.Argument{
																																																{
																																																	AssignmentExpression: javascript.AssignmentExpression{
																																																		ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																																																			MemberExpression: &javascript.MemberExpression{
																																																				MemberExpression: &javascript.MemberExpression{
																																																					PrimaryExpression: &javascript.PrimaryExpression{
																																																						IdentifierReference: props,
																																																					},
																																																				},
																																																				IdentifierName: mapt,
																																																			},
																																																			Arguments: &javascript.Arguments{
																																																				ArgumentList: []javascript.Argument{
																																																					{
																																																						AssignmentExpression: javascript.AssignmentExpression{
																																																							ArrowFunction: &javascript.ArrowFunction{
																																																								FormalParameters: &javascript.FormalParameters{
																																																									FormalParameterList: []javascript.BindingElement{
																																																										{
																																																											ArrayBindingPattern: &javascript.ArrayBindingPattern{
																																																												BindingElementList: []javascript.BindingElement{
																																																													{
																																																														SingleNameBinding: prop,
																																																													},
																																																													{
																																																														SingleNameBinding: get,
																																																													},
																																																												},
																																																											},
																																																										},
																																																									},
																																																								},
																																																								AssignmentExpression: &javascript.AssignmentExpression{
																																																									ConditionalExpression: javascript.WrapConditional(&javascript.ArrayLiteral{
																																																										ElementList: []javascript.ArrayElement{
																																																											{
																																																												AssignmentExpression: javascript.AssignmentExpression{
																																																													ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
																																																														IdentifierReference: prop,
																																																													}),
																																																												},
																																																											},
																																																											{
																																																												AssignmentExpression: javascript.AssignmentExpression{
																																																													ConditionalExpression: javascript.WrapConditional(&javascript.ObjectLiteral{
																																																														PropertyDefinitionList: []javascript.PropertyDefinition{
																																																															{
																																																																PropertyName: &javascript.PropertyName{
																																																																	LiteralPropertyName: jToken("enumerable"),
																																																																},
																																																																AssignmentExpression: trueAE,
																																																															},
																																																															{
																																																																PropertyName: &javascript.PropertyName{
																																																																	LiteralPropertyName: get,
																																																																},
																																																																AssignmentExpression: &javascript.AssignmentExpression{
																																																																	ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
																																																																		IdentifierReference: get,
																																																																	}),
																																																																},
																																																															},
																																																														},
																																																													}),
																																																												},
																																																											},
																																																										},
																																																									}),
																																																								},
																																																							},
																																																						},
																																																					},
																																																				},
																																																			},
																																																		}),
																																																	},
																																																},
																																															},
																																														},
																																													}),
																																												},
																																											},
																																										},
																																									},
																																								}),
																																							},
																																						},
																																					},
																																				},
																																			}),
																																		},
																																	},
																																},
																															}),
																														},
																													},
																												},
																											},
																										},
																									},
																								}),
																							},
																						},
																					},
																				},
																			}),
																		},
																	},
																},
															},
														},
													},
													{
														Statement: &javascript.Statement{
															Type: javascript.StatementReturn,
															ExpressionStatement: &javascript.Expression{
																Expressions: []javascript.AssignmentExpression{
																	{
																		ArrowFunction: &javascript.ArrowFunction{
																			BindingIdentifier: url,
																			AssignmentExpression: &javascript.AssignmentExpression{
																				ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																					MemberExpression: promiseResolve,
																					Arguments: &javascript.Arguments{
																						ArgumentList: []javascript.Argument{
																							{
																								AssignmentExpression: javascript.AssignmentExpression{
																									ConditionalExpression: &javascript.ConditionalExpression{
																										CoalesceExpression: &javascript.CoalesceExpression{
																											CoalesceExpressionHead: &javascript.CoalesceExpression{
																												BitwiseORExpression: javascript.WrapConditional(&javascript.CallExpression{
																													MemberExpression: importsGet,
																													Arguments: &javascript.Arguments{
																														ArgumentList: []javascript.Argument{
																															{
																																AssignmentExpression: javascript.AssignmentExpression{
																																	ConditionalExpression: wrappedURL,
																																},
																															},
																														},
																													},
																												}).LogicalORExpression.LogicalANDExpression.BitwiseORExpression,
																											},
																											BitwiseORExpression: importURL.LogicalORExpression.LogicalANDExpression.BitwiseORExpression,
																										},
																									},
																								},
																							},
																						},
																					},
																				}),
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Arguments: &javascript.Arguments{},
				}),
			}
		}
	}

	if include == nil {
		return nil
	}

	globalThis := &javascript.PrimaryExpression{
		IdentifierReference: jToken("globalThis"),
	}
	value := &javascript.PropertyName{
		LiteralPropertyName: jToken("value"),
	}
	c.statementList[0] = javascript.StatementListItem{
		Statement: &javascript.Statement{
			ExpressionStatement: &javascript.Expression{
				Expressions: []javascript.AssignmentExpression{
					{
						ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
							MemberExpression: &javascript.MemberExpression{
								MemberExpression: object,
								IdentifierName:   jToken("defineProperty"),
							},
							Arguments: &javascript.Arguments{
								ArgumentList: []javascript.Argument{
									{
										AssignmentExpression: javascript.AssignmentExpression{
											ConditionalExpression: javascript.WrapConditional(globalThis),
										},
									},
									{
										AssignmentExpression: javascript.AssignmentExpression{
											ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
												Literal: &javascript.Token{
													Token: parser.Token{
														Data: "\"include\"",
													},
												},
											}),
										},
									},
									{
										AssignmentExpression: javascript.AssignmentExpression{
											ConditionalExpression: javascript.WrapConditional(&javascript.ObjectLiteral{
												PropertyDefinitionList: []javascript.PropertyDefinition{
													{
														PropertyName:         value,
														AssignmentExpression: include,
													},
												},
											}),
										},
									},
								},
							},
						}),
					},
				},
			},
		},
	}

	return nil
}
