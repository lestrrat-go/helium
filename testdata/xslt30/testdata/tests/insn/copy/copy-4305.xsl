<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<!-- Check that in a copy of a tree, axes only navigate within the subtree that was copied -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
      xmlns:f="http://f/"
      exclude-result-prefixes="f"
      version="2.0">

    <xsl:template match="/">
      <xsl:variable name="in">
      <a>
        <b>
          <cp/>
          <c>
            <d/>
            <e/>
          </c>
          <cf/>
        </b>
      </a>
      </xsl:variable>
      <xsl:variable name="cp" select="f:copy($in//c)"/>
      <!--<xsl:variable name="cp" select="$in//c"/>-->
      <out>
      <a><xsl:value-of select="$cp/d/ancestor::*/name()"/></a>
      <a><xsl:value-of select="$cp/d/ancestor-or-self::*/name()"/></a>
      <f><xsl:value-of select="$cp/d/following::*/name()"/></f>
      <p><xsl:value-of select="$cp/d/preceding::*/name()"/></p>
      </out>
    </xsl:template>
    
    <xsl:function name="f:copy">
      <xsl:param name="node" as="node()*"/>
      <xsl:copy-of select="$node"/>
    </xsl:function>
</xsl:stylesheet>
