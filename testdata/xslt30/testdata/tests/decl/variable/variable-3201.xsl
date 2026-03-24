<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    version="2.0">

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<?spec xslt#with-param?>
  <!-- Purpose: Check propagation of params down into templates -->
  

<xsl:param name="request" select="'top'"/>

<xsl:template match="/">
  <xsl:apply-templates>
    <xsl:with-param name="p1" select="$request"/>
    <xsl:with-param name="p2" select="'root'"/>
  </xsl:apply-templates>
</xsl:template>

<xsl:template match="*">
  <xsl:param name="p1" select="'error1!'"/>
  <xsl:param name="p2" select="'error2!'"/>
	
  <xsl:copy>
    <xsl:apply-templates select="node()|@*">
      <xsl:with-param name="p1" select="$request"/>
      <xsl:with-param name="p2" select="@id"/>
    </xsl:apply-templates>
    <from>
      <xsl:call-template name="dump-values">
        <xsl:with-param name="p1" select="$request"/>
        <xsl:with-param name="p2" select="$p2"/>
      </xsl:call-template>
    </from>
  </xsl:copy>
</xsl:template>

<xsl:template match="@*">
  <xsl:param name="p1" select="'error3!'"/>
  <xsl:param name="p2" select="'error4!'"/>

  <xsl:attribute name="Value">
    <xsl:call-template name="dump-values">
      <xsl:with-param name="p1" select="$p1"/>
      <xsl:with-param name="p2" select="concat('id=', string($p2))"/>
    </xsl:call-template>
  </xsl:attribute>
</xsl:template>

<xsl:template name="dump-values">
  <xsl:param name="p1" select="'error5!'"/>
  <xsl:param name="p2" select="'error6!'"/>

  <xsl:value-of select="$p1"/>, <xsl:value-of select="$p2"/>
</xsl:template>

</xsl:stylesheet>