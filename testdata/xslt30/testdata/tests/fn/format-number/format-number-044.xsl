<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test format-number() applied to large numbers (test how good the rounding is) -->
<!-- Bug report from Pedro Christian against Saxon 7.8 -->



<xsl:template match="doc">
<out>
<xsl:value-of select="format-number(1E25,'#,######')"/>;
<xsl:value-of select="format-number(1E10,'#####################')"/>;
<xsl:value-of select="format-number(1E11,'#####################')"/>;
<xsl:value-of select="format-number(1E12,'#####################')"/>;
<xsl:value-of select="format-number(1E13,'#####################')"/>;
<xsl:value-of select="format-number(1E14,'#####################')"/>;
<xsl:value-of select="format-number(1E15,'#####################')"/>;
<xsl:value-of select="format-number(1E16,'#####################')"/>;
<xsl:value-of select="format-number(1E17,'#####################')"/>;
<xsl:value-of select="format-number(1E18,'#####################')"/>;
<xsl:value-of select="format-number(1E19,'#####################')"/>;
<xsl:value-of select="format-number(1E20,'#####################')"/>;
<xsl:value-of select="format-number(1E21,'#####################')"/>;
<xsl:value-of select="format-number(1E22,'#####################')"/>;
<xsl:value-of select="format-number(1E23,'#####################')"/>;
<xsl:value-of select="format-number(1E24,'#####################')"/>;
<xsl:value-of select="format-number(1E25,'#####################')"/>;
<xsl:value-of select="format-number(1E30,'#####################')"/>;
<xsl:value-of select="format-number(1E35,'#####################')"/>;
<xsl:value-of select="format-number(1E100,'#####################')"/>;
<xsl:value-of select="format-number(1E100 div 3,'#####################')"/>;
</out>
</xsl:template>

</xsl:stylesheet>
