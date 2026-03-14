<!DOCTYPE xsl:stylesheet SYSTEM "collation.dtd">

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="xs">

<xsl:strip-space elements="*"/>

  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'"/>


<!-- ends-with() function using explicit collation, default collation, and none -->

<xsl:template match="/">
<out>
  <th><xsl:value-of select="words/word[ends-with(., 'GH', $collation)]"/></th>
  <th><xsl:value-of select="words/word[ends-with(., 'GH')]" default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive"/></th>
  <th><xsl:value-of select="words/word[ends-with(., 'GH')]"/></th>  
</out>
</xsl:template>

</xsl:stylesheet>

