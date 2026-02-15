package jspacker

import (
	"strconv"

	"vimagination.zapto.org/javascript"
)

func namespaceImport(ns *javascript.Token, prefix string) javascript.ModuleItem {
	return javascript.ModuleItem{
		StatementListItem: &javascript.StatementListItem{
			Declaration: &javascript.Declaration{
				LexicalDeclaration: &javascript.LexicalDeclaration{
					LetOrConst: javascript.Const,
					BindingList: []javascript.LexicalBinding{
						{
							BindingIdentifier: ns,
							Initializer: &javascript.AssignmentExpression{
								ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
									IdentifierReference: jToken(prefix),
								}),
							},
						},
					},
				},
			},
		},
	}
}

func wrapVariableStatement(v *javascript.VariableStatement) javascript.ModuleItem {
	return javascript.ModuleItem{
		StatementListItem: &javascript.StatementListItem{
			Statement: &javascript.Statement{
				VariableStatement: v,
			},
		},
	}
}

func wrapDeclaration(ed *javascript.Declaration) javascript.ModuleItem {
	return javascript.ModuleItem{
		StatementListItem: &javascript.StatementListItem{
			Declaration: ed,
		},
	}
}

func wrapFunctionDeclaration(f *javascript.FunctionDeclaration) javascript.ModuleItem {
	return javascript.ModuleItem{
		StatementListItem: &javascript.StatementListItem{
			Declaration: &javascript.Declaration{
				FunctionDeclaration: f,
			},
		},
	}
}

func wrapClassDeclaration(c *javascript.ClassDeclaration) javascript.ModuleItem {
	return javascript.ModuleItem{
		StatementListItem: &javascript.StatementListItem{
			Declaration: &javascript.Declaration{
				ClassDeclaration: c,
			},
		},
	}
}

func wrapDefaultAssignment(def *javascript.Token, a *javascript.AssignmentExpression) javascript.ModuleItem {
	return javascript.ModuleItem{
		StatementListItem: &javascript.StatementListItem{
			Declaration: &javascript.Declaration{
				LexicalDeclaration: &javascript.LexicalDeclaration{
					LetOrConst: javascript.Const,
					BindingList: []javascript.LexicalBinding{
						{
							BindingIdentifier: def,
							Initializer:       a,
						},
					},
				},
			},
		},
	}
}

func wrapAssignmentExpression(ae javascript.AssignmentExpression) javascript.ModuleItem {
	return javascript.ModuleItem{
		StatementListItem: &javascript.StatementListItem{
			Statement: &javascript.Statement{
				ExpressionStatement: &javascript.Expression{
					Expressions: []javascript.AssignmentExpression{ae},
				},
			},
		},
	}
}

func importMeta(prefix, url string) javascript.LexicalBinding {
	return javascript.LexicalBinding{
		BindingIdentifier: jToken(prefix + "import"),
		Initializer: &javascript.AssignmentExpression{
			ConditionalExpression: javascript.WrapConditional(&javascript.ObjectLiteral{
				PropertyDefinitionList: []javascript.PropertyDefinition{
					{
						PropertyName: &javascript.PropertyName{
							LiteralPropertyName: jToken("url"),
						},
						AssignmentExpression: &javascript.AssignmentExpression{
							ConditionalExpression: javascript.WrapConditional(&javascript.AdditiveExpression{
								AdditiveExpression: &javascript.AdditiveExpression{
									MultiplicativeExpression: javascript.MultiplicativeExpression{
										ExponentiationExpression: javascript.ExponentiationExpression{
											UnaryExpression: javascript.UnaryExpression{
												UpdateExpression: javascript.UpdateExpression{
													LeftHandSideExpression: &javascript.LeftHandSideExpression{
														NewExpression: &javascript.NewExpression{
															MemberExpression: javascript.MemberExpression{
																PrimaryExpression: &javascript.PrimaryExpression{
																	IdentifierReference: jToken("o"),
																},
															},
														},
													},
												},
											},
										},
									},
								},
								AdditiveOperator: javascript.AdditiveAdd,
								MultiplicativeExpression: javascript.MultiplicativeExpression{
									ExponentiationExpression: javascript.ExponentiationExpression{
										UnaryExpression: javascript.UnaryExpression{
											UpdateExpression: javascript.UpdateExpression{
												LeftHandSideExpression: &javascript.LeftHandSideExpression{
													NewExpression: &javascript.NewExpression{
														MemberExpression: javascript.MemberExpression{
															PrimaryExpression: &javascript.PrimaryExpression{
																Literal: jToken(strconv.Quote(url)),
															},
														},
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
			}),
		},
	}
}

func replaceImportCall(ce *javascript.CallExpression) {
	ce.MemberExpression = &javascript.MemberExpression{
		PrimaryExpression: &javascript.PrimaryExpression{
			IdentifierReference: jToken("include"),
		},
	}
	ce.Arguments = &javascript.Arguments{
		ArgumentList: []javascript.Argument{
			{
				AssignmentExpression: *ce.ImportCall,
			},
		},
	}
	ce.ImportCall = nil
}

func locationOrigin() *javascript.StatementListItem {
	return &javascript.StatementListItem{
		Declaration: &javascript.Declaration{
			LexicalDeclaration: &javascript.LexicalDeclaration{
				LetOrConst: javascript.Const,
				BindingList: []javascript.LexicalBinding{
					{
						BindingIdentifier: jToken("o"),
						Initializer: &javascript.AssignmentExpression{
							ConditionalExpression: javascript.WrapConditional(javascript.MemberExpression{
								MemberExpression: &javascript.MemberExpression{
									PrimaryExpression: &javascript.PrimaryExpression{
										IdentifierReference: jToken("location"),
									},
								},
								IdentifierName: jToken("origin"),
							}),
						},
					},
				},
			},
		},
	}
}

func wrapArgument(arg string) javascript.Argument {
	return javascript.Argument{
		AssignmentExpression: javascript.AssignmentExpression{
			ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
				Literal: jToken(strconv.Quote(arg)),
			}),
		},
	}
}

func wrapMemberIdentifier(id string, in *javascript.Token) javascript.MemberExpression {
	return javascript.MemberExpression{
		MemberExpression: &javascript.MemberExpression{
			PrimaryExpression: &javascript.PrimaryExpression{
				IdentifierReference: jToken(id),
			},
		},
		IdentifierName: in,
	}
}

func wrapIncludeCall(ident *javascript.Token, args []javascript.Argument) javascript.ModuleItem {
	return javascript.ModuleItem{
		StatementListItem: &javascript.StatementListItem{
			Declaration: &javascript.Declaration{
				LexicalDeclaration: &javascript.LexicalDeclaration{
					LetOrConst: javascript.Const,
					BindingList: []javascript.LexicalBinding{
						{
							BindingIdentifier: ident,
							Initializer: &javascript.AssignmentExpression{
								ConditionalExpression: javascript.WrapConditional(&javascript.UnaryExpression{
									UnaryOperators: []javascript.UnaryOperatorComments{{UnaryOperator: javascript.UnaryAwait}},
									UpdateExpression: javascript.UpdateExpression{
										LeftHandSideExpression: &javascript.LeftHandSideExpression{
											CallExpression: &javascript.CallExpression{
												MemberExpression: &javascript.MemberExpression{
													PrimaryExpression: &javascript.PrimaryExpression{
														IdentifierReference: jToken("include"),
													},
												},
												Arguments: &javascript.Arguments{
													ArgumentList: args,
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
	}
}

func wrapIncludeAllCall(importObjectBindings []javascript.BindingElement, importURLsArrayE []javascript.ArrayElement) javascript.ModuleItem {
	return javascript.ModuleItem{
		StatementListItem: &javascript.StatementListItem{
			Declaration: &javascript.Declaration{
				LexicalDeclaration: &javascript.LexicalDeclaration{
					LetOrConst: javascript.Const,
					BindingList: []javascript.LexicalBinding{
						{
							ArrayBindingPattern: &javascript.ArrayBindingPattern{
								BindingElementList: importObjectBindings,
							},
							Initializer: &javascript.AssignmentExpression{
								ConditionalExpression: javascript.WrapConditional(&javascript.UnaryExpression{
									UnaryOperators: []javascript.UnaryOperatorComments{{UnaryOperator: javascript.UnaryAwait}},
									UpdateExpression: javascript.UpdateExpression{
										LeftHandSideExpression: &javascript.LeftHandSideExpression{
											CallExpression: &javascript.CallExpression{
												MemberExpression: &javascript.MemberExpression{
													MemberExpression: &javascript.MemberExpression{
														PrimaryExpression: &javascript.PrimaryExpression{
															IdentifierReference: jToken("Promise"),
														},
													},
													IdentifierName: jToken("all"),
												},
												Arguments: &javascript.Arguments{
													ArgumentList: []javascript.Argument{
														{
															AssignmentExpression: javascript.AssignmentExpression{
																ConditionalExpression: javascript.WrapConditional(&javascript.CallExpression{
																	MemberExpression: &javascript.MemberExpression{
																		MemberExpression: &javascript.MemberExpression{
																			PrimaryExpression: &javascript.PrimaryExpression{
																				ArrayLiteral: &javascript.ArrayLiteral{
																					ElementList: importURLsArrayE,
																				},
																			},
																		},
																		IdentifierName: jToken("map"),
																	},
																	Arguments: &javascript.Arguments{
																		ArgumentList: []javascript.Argument{
																			{
																				AssignmentExpression: javascript.AssignmentExpression{
																					ConditionalExpression: javascript.WrapConditional(&javascript.PrimaryExpression{
																						IdentifierReference: jToken("include"),
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
									},
								}),
							},
						},
					},
				},
			},
		},
	}
}
