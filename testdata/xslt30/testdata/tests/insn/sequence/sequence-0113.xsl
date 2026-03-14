<?xml version="1.0"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:s="http://s.com/" exclude-result-prefixes="s">

  <!-- PURPOSE:   Test navigation from a parentless element -->
  <xsl:template match="doc">
    <xsl:variable name="x" as="element()">
      <a a1="a1" xmlns:a="http://a.uri/">
        <b b1="b1">
          <c/>
          <d/>
        </b>
      </a>
    </xsl:variable>
    <zzz>
      <a>
        <xsl:value-of select="count($x/a)"/>
      </a>
      <b>
        <xsl:value-of select="count($x/b)"/>
      </b>
      <c>
        <xsl:value-of select="count($x/c)"/>
      </c>
      <all>
        <xsl:value-of select="count($x//*)"/>
      </all>
      <xsl:for-each select="$x//c">
        <anc>
          <xsl:value-of select="for $i in ancestor::node() return name($i)" separator=","/>
        </anc>
        <att>
          <xsl:value-of select="ancestor::a/@a1"/>
        </att>
        <ns>
          <xsl:value-of select="s:sort(namespace::*)" separator=","/>
        </ns>
      </xsl:for-each>
    </zzz>
  </xsl:template>

  <xsl:function name="s:sort">
    <xsl:param name="seq"/>
    <xsl:perform-sort select="$seq">
      <xsl:sort/>
    </xsl:perform-sort>
  </xsl:function>

</xsl:stylesheet>
