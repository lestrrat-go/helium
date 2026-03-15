<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="xs">

<xsl:strip-space elements="*"/>

  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'"/>


<!-- Sort strings in descending order using a case-blind collation -->

<xsl:template match="/">
<out>
  <xsl:for-each select="words/word">
    <xsl:sort select="." collation="{$collation}" order="descending"/>
    <w><xsl:value-of select="."/></w>
  </xsl:for-each>
</out>
</xsl:template>

</xsl:stylesheet>

