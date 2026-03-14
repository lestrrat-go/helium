<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all"
  xmlns:mf="http://example.com/mf"
  expand-text="yes">
  
  <xsl:function name="mf:wrap" as="node()*">
    <xsl:param name="nodes" as="node()*"/>
    <xsl:param name="tag" as="xs:string"/>
    <xsl:for-each-group select="$nodes" group-starting-with="e[@tag = $tag]">
      <xsl:choose>
        <xsl:when test="not(self::e[@tag = $tag])">
          <xsl:apply-templates select="current-group()"/>
        </xsl:when>
        <xsl:otherwise>
          <xsl:for-each-group select="tail(current-group())" group-ending-with="e[@tag = '/' || $tag]">
            <xsl:choose>
              <xsl:when test="current-group()[last()][self::e][@tag = '/' || $tag]">
                <xsl:element name="{$tag}">
                  <xsl:apply-templates select="current-group()[position() lt last()]"/>
                </xsl:element>
              </xsl:when>
              <xsl:otherwise>
                <!--<xsl:apply-templates select="current-group()"/>-->
              </xsl:otherwise>
            </xsl:choose>
          </xsl:for-each-group>
        </xsl:otherwise>
      </xsl:choose>
    </xsl:for-each-group>
  </xsl:function>
  
  <xsl:function name="mf:wrap" as="node()*">
    <xsl:param name="nodes" as="node()*"/>
    <xsl:sequence
      select="fold-left(distinct-values($nodes[self::e/@tag[not(starts-with(., '/'))]]/@tag), $nodes, mf:wrap#2)"/>
  </xsl:function>
  
  <xsl:mode on-no-match="shallow-copy"/>
  
  <xsl:template match="*[e[@tag]]">
    <xsl:copy>
      <xsl:apply-templates select="@*"/>
      <xsl:sequence select="mf:wrap(node())"/>
    </xsl:copy>
  </xsl:template>
  
</xsl:stylesheet>