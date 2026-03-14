<?xml version="1.0"?> 
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test for recursion with xsl:call-template. -->
<!-- <xsl:param name="pvar2" select="'stylesheet-var'"/> -->
 
<xsl:template match="/doc">
   <out>
      <xsl:variable name="ResultTreeFragTest" select="name(.)"/> 
      <xsl:call-template name="ntmp1">
        <xsl:with-param name="pvar1" select="$ResultTreeFragTest"/>
      </xsl:call-template>
   </out>
</xsl:template>

<xsl:template name="ntmp1">
    <xsl:param name="pvar1">def-text-1</xsl:param>
    <xsl:param name="pvar2">def-text-2</xsl:param>
    <xsl:value-of select="$pvar1"/><xsl:text>,</xsl:text>
    <xsl:value-of select="$pvar2"/><xsl:text>,</xsl:text>
    <xsl:apply-templates select="*">
      <xsl:with-param name="pvar1" select="$pvar1"/>
    </xsl:apply-templates>
</xsl:template>

<xsl:template match="*">
    <xsl:param name="pvar1">def-text-1</xsl:param>
    <xsl:call-template name="ntmp1">
      <xsl:with-param name="pvar1" select="$pvar1"/>
      <xsl:with-param name="pvar2" select="name(.)"/>
    </xsl:call-template>
</xsl:template>
  
</xsl:stylesheet>