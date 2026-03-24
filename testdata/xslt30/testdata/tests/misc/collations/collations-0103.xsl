<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="xs">

<xsl:strip-space elements="*"/>

  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive'"/>


<!-- Grouping using a case-blind collation -->

<xsl:template match="/">
<out>
  <xsl:for-each-group select="words/word" group-by="." collation="{$collation}">
    <group>
      <xsl:copy-of select="current-group()"/>
    </group>
  </xsl:for-each-group>
</out>
</xsl:template>

</xsl:stylesheet>

