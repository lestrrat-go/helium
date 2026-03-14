<?xml version="1.0"?>

<!-- PURPOSE: test union operation applied to namespace nodes -->
  <?spec xpath#combining_seq?>

<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:template match="/">
    <out>
      <A>
        <xsl:for-each select="//node()|//namespace::*">
          <xsl:sort select="name()"/>
          <xsl:sort select="."/>
          <xsl:call-template name="show"/>
        </xsl:for-each>
        <a><xsl:value-of select="count(//node()|//namespace::*)"/> combined nodes</a>
      </A>

      <B>
        <xsl:for-each select="//namespace::*">
          <xsl:sort select="name()"/>
          <xsl:sort select="."/>
          <xsl:call-template name="show"/>
        </xsl:for-each>
        <b><xsl:value-of select="count(//namespace::*)"/> namespace nodes</b>
      </B>

      <C>
        <xsl:for-each select="//node()">
          <xsl:call-template name="show"/>
        </xsl:for-each>
        <c><xsl:value-of select="count(//node())"/> node()s</c>
      </C>
    </out>
  </xsl:template>

  <xsl:template name="show">
    <z>
      <xsl:value-of select="name()"/>
    </z>
  </xsl:template>

</xsl:transform>
