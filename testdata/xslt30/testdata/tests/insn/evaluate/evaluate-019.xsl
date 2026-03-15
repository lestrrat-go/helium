<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" 
				xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
				xmlns:xs="http://www.w3.org/2001/XMLSchema"
				xmlns="http://www.w3.org/1999/xhtml"
				exclude-result-prefixes=" xs" >
				
<!-- test that xsl:evaluate doesn't use the default namespace (bug in 8.7 and previous releases)-->
<!-- formerly saxon096 -->
				
	<xsl:template match="/">
		<html>
			<head>
				<meta name="Contributor" content="Fred"/>
				<title>test</title>
			</head>
			<body>
				<p><xsl:text>normal xpath/xslt : </xsl:text> <xsl:value-of select="/document/data"/></p>
				<p><xsl:text>evaluate : </xsl:text> <xsl:evaluate xpath="'/document/data/string()'" context-item="."/></p>
				<p><xsl:text>evaluate without namespaces : </xsl:text> <xsl:evaluate xpath="'/*[1]/*[1]/string()'" context-item="."/></p>
			</body>
		</html>
	</xsl:template>
	
</xsl:stylesheet>
