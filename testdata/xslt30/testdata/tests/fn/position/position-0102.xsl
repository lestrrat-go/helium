<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs">

  <xsl:template name="go">
  <out>
    <xsl:variable name="t" as="xs:string+" select="'X', 'X'"/>
    <xsl:variable name="l" as="xs:integer+" select="0, 3[$t[2] != 'A']"/>

    <xsl:for-each select="$l">
      <xsl:if test="position() != last()">
        <xsl:variable name="i" as="xs:integer" select="position()"/>
        <xsl:variable name="p" as="xs:integer" select=". + 1"/>
        <xsl:variable name="n" as="xs:integer" select="$l[$i + 1]"/>

        <xsl:sequence select="subsequence($t, $p, $n - $p)"/>
      </xsl:if>
    </xsl:for-each>
  </out>  
  </xsl:template>

</xsl:stylesheet>

