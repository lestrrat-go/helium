<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- DOCUMENT: http://www.w3.org/TR/xpath -->
  <!-- PURPOSE: Test optimisation of recursive scan of a node-set -->
  <?spec xslt#named-templates?>
  
<xsl:template match="doc">
    <out>
          <xsl:call-template name="t">
            <xsl:with-param name="ns" select="//item"/>
          </xsl:call-template>
    </out>
</xsl:template>

<xsl:template name="t">
    <xsl:param name="ns"/>
    <xsl:choose>
    <xsl:when test="not($ns)">0</xsl:when>
    <xsl:otherwise>
        <xsl:variable name="remainder">
        <xsl:call-template name="t">
            <xsl:with-param name="ns" select="$ns[position()>1]"/>
        </xsl:call-template>
        </xsl:variable>
        <xsl:value-of select="$remainder + ($ns[1]/@price * $ns[1]/@qty)"/>
    </xsl:otherwise>
    </xsl:choose>
</xsl:template>

</xsl:stylesheet>
