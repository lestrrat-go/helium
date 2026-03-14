<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test = and !=, comparing RTF to string. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:variable name="numericRTF">
  <xsl:value-of select="/doc/num"/>
</xsl:variable>

<xsl:template match="doc">
  <out>
    <en><xsl:value-of select="$numericRTF=17"/></en>
    <nn><xsl:value-of select="$numericRTF!=17"/></nn>
    <ne><xsl:value-of select="17=$numericRTF"/></ne>
    <nn><xsl:value-of select="17!=$numericRTF"/></nn>
  </out>
</xsl:template>

</xsl:stylesheet>