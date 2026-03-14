<?xml version='1.0' encoding='utf-8' ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  version="2.0">

<?spec xslt#with-param?>
  
  <!-- Purpose: test parameters call by reference and by value -->
 
<xsl:output method="xml" indent="no"/>

<xsl:template match="/">
  <out><xsl:text>&#10;</xsl:text>
    <xsl:variable name="foo" select="/data/point"/>

    <list m="ByReference">
      <xsl:call-template name="AllOrNothing">
        <xsl:with-param name="path" select="$foo"/>
      </xsl:call-template>
    </list><xsl:text>&#10;</xsl:text>
    <list m="ByValue">
      <xsl:call-template name="AllOrNothing">
        <xsl:with-param name="path" select="/data/point"/>
      </xsl:call-template>
    </list><xsl:text>&#10;</xsl:text>
    <list m="ByReference">
      <xsl:call-template name="AlmostAndNothing">
        <xsl:with-param name="path" select="$foo"/>
      </xsl:call-template>
    </list><xsl:text>&#10;</xsl:text>
    <list m="ByValue">
      <xsl:call-template name="AlmostAndNothing">
        <xsl:with-param name="path" select="/data/point"/>
      </xsl:call-template>
    </list><xsl:text>&#10;</xsl:text>
  </out>
</xsl:template>

<xsl:template name="AllOrNothing">
  <xsl:param name="path"/>

  <xsl:for-each select="/data/point">
    <xsl:variable name="pos" select="position()"/>

    <xsl:for-each select="$path[$pos]">
      <xsl:value-of select="."/><xsl:text>/</xsl:text>
    </xsl:for-each>
  </xsl:for-each>
</xsl:template>

<xsl:template name="AlmostAndNothing">
  <xsl:param name="path"/>

  <xsl:for-each select="$path">
    <xsl:variable name="pos" select="position()"/>

    <xsl:for-each select="$path[$pos]">
      <xsl:value-of select="."/><xsl:text>-</xsl:text>
    </xsl:for-each>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>