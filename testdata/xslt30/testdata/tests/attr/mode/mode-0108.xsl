<?xml version="1.0"?> 
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE:  Test apply-templates when context node is a temporary tree  -->
  <xsl:template match="/">
    <out>
      <xsl:for-each select="$x">
        <xsl:apply-templates mode="a"/>
      </xsl:for-each>
    </out>
  </xsl:template>

  <xsl:template match="/" mode="a">
    <tree>
      <xsl:apply-templates mode="a"/>
    </tree>
  </xsl:template>

  <xsl:template match="x" mode="a">
    <x/>
  </xsl:template>

  <xsl:variable name="x">
    <x/>
  </xsl:variable>

</xsl:stylesheet>
