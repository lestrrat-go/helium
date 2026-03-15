<?xml version="1.0" encoding="ISO-8859-1"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <!-- DOCUMENT: http://www.w3.org/TR/xpath -->
  <!-- SECTION:  4.1 Node Set Functions. -->
  <!-- PURPOSE:   Test of recursive '[position()>1]' optimizations. -->

  <!-- this is a variant of pos63 in which the expression $nodes[1] is replaced
     by a numeric expression whose value (and indeed type) is not known statically.
     This takes a different (and difficult) path through the FilterExpression code -->

  <xsl:strip-space elements="*"/>
  <xsl:param name="p" select="1"/>
  
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
        <xsl:copy-of select="$nodes[$p]"/>
        <xsl:call-template name="recurse">
          <xsl:with-param name="nodes" select="$nodes[position() &gt; 1]"/>
        </xsl:call-template>
      </xsl:when>
      <xsl:otherwise/>
    </xsl:choose>
  </xsl:template>
  
</xsl:stylesheet>
