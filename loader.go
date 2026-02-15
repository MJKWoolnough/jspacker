package jspacker

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"

	"vimagination.zapto.org/javascript"
	"vimagination.zapto.org/parser"
)

func jToken(data string) *javascript.Token {
	return &javascript.Token{Token: parser.Token{Data: data}}
}

func (c *config) makeLoader() error {
	bes := make([]javascript.BindingElement, 0, len(c.filesDone))
	obs := make([]javascript.ArrayElement, 0, len(c.filesDone))

	for _, file := range sortedMap(c.filesDone) {
		bes = append(bes, javascript.BindingElement{
			SingleNameBinding: jToken(file.prefix),
		})
		fields := make([]javascript.ArrayElement, 0, len(file.exports))

		for binding := range sortedMap(file.exports) {
			b := file.resolveExport(binding)

			if b == nil {
				return fmt.Errorf("error resolving export %s (%s): %w", binding, file.url, ErrInvalidExport)
			}

			fields = append(fields, javascript.ArrayElement{
				AssignmentExpression: javascript.AssignmentExpression{
					ConditionalExpression: javascript.WrapConditional(&javascript.ArrayLiteral{
						ElementList: []javascript.ArrayElement{
							{
								AssignmentExpression: javascript.AssignmentExpression{
									ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
										Literal: jToken(strconv.Quote(binding)),
									}),
								},
							},
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
						},
					}),
				},
			})
		}

		obs = append(obs, javascript.ArrayElement{
			AssignmentExpression: javascript.AssignmentExpression{
				ConditionalExpression: javascript.WrapConditional(&javascript.ArrayLiteral{
					ElementList: fields,
				}),
			},
		})
	}

	c.moduleItems = slices.Insert(c.moduleItems, 0, javascript.ModuleItem{
		StatementListItem: &javascript.StatementListItem{
			Declaration: &javascript.Declaration{
				LexicalDeclaration: &javascript.LexicalDeclaration{
					LetOrConst: javascript.Const,
					BindingList: []javascript.LexicalBinding{
						{
							ArrayBindingPattern: &javascript.ArrayBindingPattern{
								BindingElementList: bes,
							},
							Initializer: &javascript.AssignmentExpression{
								ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
									MemberExpression: &javascript.MemberExpression{
										MemberExpression: &javascript.MemberExpression{
											PrimaryExpression: &javascript.PrimaryExpression{
												ArrayLiteral: &javascript.ArrayLiteral{
													ElementList: obs,
												},
											},
										},
										IdentifierName: jToken("map"),
									},
									Arguments: &javascript.Arguments{
										ArgumentList: []javascript.Argument{
											{
												AssignmentExpression: javascript.AssignmentExpression{
													ArrowFunction: &javascript.ArrowFunction{
														BindingIdentifier: jToken("props"),
														AssignmentExpression: &javascript.AssignmentExpression{
															ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																MemberExpression: &javascript.MemberExpression{
																	MemberExpression: &javascript.MemberExpression{
																		PrimaryExpression: &javascript.PrimaryExpression{
																			IdentifierReference: jToken("Object"),
																		},
																	},
																	IdentifierName: jToken("freeze"),
																},
																Arguments: &javascript.Arguments{
																	ArgumentList: []javascript.Argument{
																		{
																			AssignmentExpression: javascript.AssignmentExpression{
																				ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																					MemberExpression: &javascript.MemberExpression{
																						MemberExpression: &javascript.MemberExpression{
																							PrimaryExpression: &javascript.PrimaryExpression{
																								IdentifierReference: jToken("Object"),
																							},
																						},
																						IdentifierName: jToken("defineProperties"),
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
																											MemberExpression: &javascript.MemberExpression{
																												PrimaryExpression: &javascript.PrimaryExpression{
																													IdentifierReference: jToken("Object"),
																												},
																											},
																											IdentifierName: jToken("fromEntries"),
																										},
																										Arguments: &javascript.Arguments{
																											ArgumentList: []javascript.Argument{
																												{
																													AssignmentExpression: javascript.AssignmentExpression{
																														ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																															MemberExpression: &javascript.MemberExpression{
																																MemberExpression: &javascript.MemberExpression{
																																	PrimaryExpression: &javascript.PrimaryExpression{
																																		IdentifierReference: jToken("props"),
																																	},
																																},
																																IdentifierName: jToken("map"),
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
																																										SingleNameBinding: jToken("prop"),
																																									},
																																									{
																																										SingleNameBinding: jToken("get"),
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
																																										IdentifierReference: jToken("prop"),
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
																																												AssignmentExpression: &javascript.AssignmentExpression{
																																													ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
																																														Literal: jToken("true"),
																																													}),
																																												},
																																											},
																																											{
																																												PropertyName: &javascript.PropertyName{
																																													LiteralPropertyName: jToken("get"),
																																												},
																																												AssignmentExpression: &javascript.AssignmentExpression{
																																													ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
																																														IdentifierReference: jToken("get"),
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
	})

	if !c.bare || c.parseDynamic {
		imports := make([]javascript.ArrayElement, 0, len(c.filesDone))

		for url, file := range sortedMap(c.filesDone) {
			imports = append(imports, javascript.ArrayElement{
				AssignmentExpression: javascript.AssignmentExpression{
					ConditionalExpression: javascript.WrapConditional(&javascript.ArrayLiteral{
						ElementList: []javascript.ArrayElement{
							{
								AssignmentExpression: javascript.AssignmentExpression{
									ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
										Literal: jToken(strconv.Quote(url)),
									}),
								},
							},
							{
								AssignmentExpression: javascript.AssignmentExpression{
									ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
										IdentifierReference: jToken(file.prefix),
									}),
								},
							},
						},
					}),
				},
			})
		}

		c.moduleItems = slices.Insert(c.moduleItems, 1, javascript.ModuleItem{
			StatementListItem: &javascript.StatementListItem{
				Statement: &javascript.Statement{
					ExpressionStatement: &javascript.Expression{
						Expressions: []javascript.AssignmentExpression{
							{
								ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
									MemberExpression: &javascript.MemberExpression{
										MemberExpression: &javascript.MemberExpression{
											PrimaryExpression: &javascript.PrimaryExpression{
												IdentifierReference: jToken("Object"),
											},
										},
										IdentifierName: jToken("defineProperty"),
									},
									Arguments: &javascript.Arguments{
										ArgumentList: []javascript.Argument{
											{
												AssignmentExpression: javascript.AssignmentExpression{
													ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
														IdentifierReference: jToken("globalThis"),
													}),
												},
											},
											{
												AssignmentExpression: javascript.AssignmentExpression{
													ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
														Literal: jToken("include"),
													}),
												},
											},
											{
												AssignmentExpression: javascript.AssignmentExpression{
													ConditionalExpression: javascript.WrapConditional(&javascript.ObjectLiteral{
														PropertyDefinitionList: []javascript.PropertyDefinition{
															{
																PropertyName: &javascript.PropertyName{
																	LiteralPropertyName: jToken("value"),
																},
																AssignmentExpression: &javascript.AssignmentExpression{
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
																															BindingIdentifier: jToken("imports"),
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
																																					ConditionalExpression: javascript.WrapConditional(&javascript.ArrayLiteral{
																																						ElementList: imports,
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
																																BindingIdentifier: jToken("url"),
																																AssignmentExpression: &javascript.AssignmentExpression{
																																	ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																																		MemberExpression: &javascript.MemberExpression{
																																			MemberExpression: &javascript.MemberExpression{
																																				PrimaryExpression: &javascript.PrimaryExpression{
																																					IdentifierReference: jToken("Promise"),
																																				},
																																				IdentifierName: jToken("resolve"),
																																			},
																																		},
																																		Arguments: &javascript.Arguments{
																																			ArgumentList: []javascript.Argument{
																																				{
																																					AssignmentExpression: javascript.AssignmentExpression{
																																						ConditionalExpression: &javascript.ConditionalExpression{
																																							CoalesceExpression: &javascript.CoalesceExpression{
																																								CoalesceExpressionHead: &javascript.CoalesceExpression{
																																									BitwiseORExpression: javascript.WrapConditional(&javascript.CallExpression{
																																										MemberExpression: &javascript.MemberExpression{
																																											MemberExpression: &javascript.MemberExpression{
																																												PrimaryExpression: &javascript.PrimaryExpression{
																																													IdentifierReference: jToken("imports"),
																																												},
																																											},
																																											IdentifierName: jToken("get"),
																																										},
																																										Arguments: &javascript.Arguments{
																																											ArgumentList: []javascript.Argument{
																																												{
																																													AssignmentExpression: javascript.AssignmentExpression{
																																														ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
																																															IdentifierReference: jToken("url"),
																																														}),
																																													},
																																												},
																																											},
																																										},
																																									}).LogicalORExpression.LogicalANDExpression.BitwiseORExpression,
																																								},
																																								BitwiseORExpression: javascript.WrapConditional(&javascript.CallExpression{
																																									ImportCall: &javascript.AssignmentExpression{
																																										ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
																																											IdentifierReference: jToken("url"),
																																										}),
																																									},
																																								}).LogicalORExpression.LogicalANDExpression.BitwiseORExpression,
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
		})
	}

	return nil
}

func sortedMap[K cmp.Ordered, V any](m map[K]V) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		keys := slices.Collect(maps.Keys(m))

		slices.Sort(keys)

		for _, k := range keys {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}
