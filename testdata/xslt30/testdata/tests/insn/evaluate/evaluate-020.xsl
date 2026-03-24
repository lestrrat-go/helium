<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" 
				xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
				xmlns:xs="http://www.w3.org/2001/XMLSchema"
				xmlns="http://www.w3.org/1999/xhtml"
				exclude-result-prefixes="xs">
				
				<!-- test that xsl:evaluate does use the default namespace from the source document -->
				<!-- formerly saxon097 -->
	
	<xsl:template match="/">
		<html>
			<head>
				<meta name="Contributor" content="Fred"/>
				<title>test</title>
			</head>
			<body>
				<p><xsl:text>evaluate-node : </xsl:text>
					<xsl:variable name="path" select="/d:document/d:path" xmlns:d="http://saxon097.uri/"/>
				    <xsl:for-each select="$path"> 
				     <xsl:evaluate xpath="string($path)"
				                   context-item="."
				                   base-uri="base-uri($path)"
				                   namespace-context="$path"
				                   as="xs:string"/>
				   </xsl:for-each>
				   </p>
			</body>
		</html>
	</xsl:template>
	
</xsl:stylesheet>
