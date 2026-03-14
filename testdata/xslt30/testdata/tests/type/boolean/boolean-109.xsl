<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test =, !=, and not, comparing node-set to number. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:variable name="x" select="avj/good/*"/>
    <e><xsl:value-of select="$x=34"/></e>
    <ne><xsl:value-of select="not($x=34)"/></ne>
    <n><xsl:value-of select="$x!=34"/></n>
    <nn><xsl:value-of select="not($x!=34)"/></nn><xsl:text>
</xsl:text><!-- Now reverse the order of the comparands -->
    <e><xsl:value-of select="34=$x"/></e>
    <ne><xsl:value-of select="not(34=$x)"/></ne>
    <n><xsl:value-of select="34!=$x"/></n>
    <nn><xsl:value-of select="not(34!=$x)"/></nn>
  </out>
</xsl:template>

</xsl:stylesheet>