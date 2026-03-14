<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE:   Test of recursive '[position()>1]' optimizations. -->
  <xsl:strip-space elements="*"/>
  
  <xsl:template match="doc">
    <out>
      <xsl:call-template name="recurse">
        <xsl:with-param name="nodes" select="*"/>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="recurse">
    <xsl:param name="nodes"/>
    <xsl:choose>
      <xsl:when test="$nodes">
        <xsl:copy-of select="$nodes[1]"/>
        <xsl:call-template name="recurse">
          <xsl:with-param name="nodes" select="$nodes[position() &gt; 1]"/>
        </xsl:call-template>
      </xsl:when>
      <xsl:otherwise/>
    </xsl:choose>
  </xsl:template>
</xsl:stylesheet>
