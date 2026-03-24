<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <xsl:output method="xml" encoding="UTF-8" indent="no"/>
    <xsl:template match="doc">
<x>
  <!--Testing Round#2 for xs:Integer -->
   <?spec fo#func-round?>
    <a><xsl:value-of select="round(0,0)"/></a>    
    <b><xsl:value-of select="round(8450,-2)"/></b>
    <c><xsl:value-of select="round(-8450,-2)"/></c>
    <d><xsl:value-of select="round(8492, 2)"/></d>
    <e><xsl:value-of select="round(77,0)"/></e>
    <f><xsl:value-of select="round(8490,-2)"/></f>
    <g><xsl:value-of select="round(8352,-2)"/></g>
    <h><xsl:value-of select="round(8462,-2)"/></h>
    <i><xsl:value-of select="round(8400,-2)"/></i>
    <j><xsl:value-of select="round(8449,-2)"/></j>
    <k><xsl:value-of select="round(-8490,-2)"/></k>
    <l><xsl:value-of select="round(-8352,-2)"/></l>
    <m><xsl:value-of select="round(-8462,-2)"/></m>
    <n><xsl:value-of select="round(-8400,-2)"/></n>
    <o><xsl:value-of select="round(-8449,-2)"/></o>
    <p><xsl:value-of select="round(1234567890123456, -15)"/></p>
    <q><xsl:value-of select="round(1534567890123454, -15)"/></q>
    <r><xsl:value-of select="round(123456789012345789011,-1)"/></r>
    <s><xsl:value-of select="round(123456789012345789045,-1)"/></s>
    <t><xsl:value-of select="round(-1234567890123456, -15)"/></t>
    <u><xsl:value-of select="round(-1534567890123454, -15)"/></u>
    <v><xsl:value-of select="round(-123456789012345789011,-1)"/></v>
    <w><xsl:value-of select="round(-123456789012345789045,-1)"/></w>
    <out>
        <xsl:value-of select="round(sum(for $e in a return number($e)),-2)"/>   
    </out>    
</x>
    </xsl:template>
</xsl:stylesheet>
