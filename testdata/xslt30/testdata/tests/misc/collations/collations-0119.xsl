<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="xs">

<xsl:strip-space elements="*"/>

  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'"/>


<!-- min() function using explicit collation, default collation, and none -->

<xsl:template match="/">
<out>
  <th><xsl:value-of select="min(words/word[position()=1 to 6]/string(), $collation)"/></th>
  <th><xsl:value-of select="min(words/word[position()=1 to 6]/string())" default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"/></th>
  <th><xsl:value-of select="min(words/word[position()=1 to 6]/string())"/></th>  
</out>
</xsl:template>

</xsl:stylesheet>

