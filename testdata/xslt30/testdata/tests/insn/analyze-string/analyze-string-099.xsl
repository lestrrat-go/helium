<xsl:stylesheet version="3.0" 
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 exclude-result-prefixes="#all">
 
 <!-- A test for non-greedy matching -->
 
 <xsl:template match="/" name="xsl:initial-template">
  <out>
   <xsl:variable name="text" as="xs:string">'A = "a" OR B = "b"'</xsl:variable> 
  
  <xsl:analyze-string regex='"(%%|.)*?"' select="$text">
   <xsl:matching-substring>
    <match>
     <xsl:sequence select="regex-group(0)"/>
    </match>
   </xsl:matching-substring>
  </xsl:analyze-string>
  </out>
 </xsl:template>
 
</xsl:stylesheet>