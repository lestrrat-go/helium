<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Bug report from Evan Lenz against Saxon 9.0.0.6, 6 June 2008 -->

<xsl:output method="xml"/>

<xsl:variable name="list" as="element()*">
  <item num="1">One</item>
  <item num="2">Two</item>
  <item num="3">Three</item>
</xsl:variable>

<xsl:template match="/">
  <xsl:variable name="highest-num" select="max($list/@num)"/>

  <!-- The following results in: "XPTY0004: Cannot compare xs:double to xs:double" -->
  <out><xsl:value-of select="$list[@num = $highest-num]"/></out>

  <!-- However, this alternative implementation (without using a
variable) works fine:
  <xsl:value-of select="$list[@num = max($list/@num)]"/>
  -->
</xsl:template>

</xsl:stylesheet>
