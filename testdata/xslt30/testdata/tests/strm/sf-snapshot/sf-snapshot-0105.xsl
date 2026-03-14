<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
 xmlns:f="http://example.com/functions" 
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 exclude-result-prefixes="f xs">
 

<xsl:template name="main">
 <xsl:source-document streamable="yes" href="sf-snapshot-01.xml">
   <out>
    <xsl:sequence select="snapshot(/a/b/c/d/e)"/>
   </out>
 </xsl:source-document>  
</xsl:template> 
 

</xsl:stylesheet>