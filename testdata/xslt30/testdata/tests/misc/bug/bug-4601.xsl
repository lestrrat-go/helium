<xsl:transform
         xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
         xmlns:bib="http://greenbytes.de/2000/xmlns/bibliography"
         xmlns="http://www.w3.org/1999/xhtml"
         exclude-result-prefixes="bib"
         version="2.0"
 >
 
 <?spec xpath#node-tests?>
 <!-- BUG 6.1/010: //abc:xyz returns no nodes when used with tinytree model -->

 <xsl:output encoding="UTF-8" omit-xml-declaration="yes" />

 <xsl:template match="/">
 	<c><xsl:value-of select="count(//bib:def)" /> def nodes in bib namespace.</c>
 </xsl:template>

 </xsl:transform>
