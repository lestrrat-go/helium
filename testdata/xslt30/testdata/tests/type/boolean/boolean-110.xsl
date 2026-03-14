<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test = and !=, comparing RTF to string. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:variable name="stringRTF">
  <xsl:value-of select="/doc/str"/>
</xsl:variable>

<xsl:template match="doc">
  <out>
    <es><xsl:value-of select="$stringRTF='found'"/></es>
    <ns><xsl:value-of select="$stringRTF!='found'"/></ns>
    <se><xsl:value-of select="'found'=$stringRTF"/></se>
    <sn><xsl:value-of select="'found'!=$stringRTF"/></sn>
  </out>
</xsl:template>

</xsl:stylesheet>