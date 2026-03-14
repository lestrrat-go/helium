<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="xs">

<xsl:strip-space elements="*"/>
  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'"/>


<!-- substring-after() function using explicit collation, default collation, and none -->

<xsl:template match="/">
<out>
  <th><xsl:value-of select="words/word/substring-after(., 'OU', $collation)" separator="|"/></th>
  <th><xsl:value-of select="words/word/substring-after(., 'OU')" default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive" separator="|"/></th>
  <th><xsl:value-of select="words/word/substring-after(., 'OU')" separator="|"/></th>  
</out>
</xsl:template>

</xsl:stylesheet>

