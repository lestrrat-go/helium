<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="xs">

<xsl:strip-space elements="*"/>

  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'"/>


<!-- eq operator using default collation to compare two equal words -->

  <xsl:template match="/" default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive">
<out>
  <eq><xsl:value-of select="words/word[3] eq words/word[10]"/></eq>
  <ne><xsl:value-of select="words/word[3] ne words/word[10]"/></ne>  
  <gt><xsl:value-of select="words/word[3] gt words/word[10]"/></gt>
  <lt><xsl:value-of select="words/word[3] lt words/word[10]"/></lt>  
  <ge><xsl:value-of select="words/word[3] ge words/word[10]"/></ge>
  <le><xsl:value-of select="words/word[3] le words/word[10]"/></le>    
</out>
</xsl:template>

</xsl:stylesheet>

