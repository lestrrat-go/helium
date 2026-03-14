<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:dbf="http://nwalsh.com/docbook/xsl/functions" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  version="2.0">

  <xsl:function name="dbf:user-node-matches" as="xs:boolean">
    <xsl:param name="template-node" as="element()"/>
    <xsl:param name="document-node" as="element()"/>
    <xsl:value-of select="true()"/>
  </xsl:function>

  <xsl:function name="dbf:node-matches1" as="xs:boolean">
    <xsl:param name="template-node" as="element()"/>
    <xsl:param name="document-node" as="element()"/>

    <xsl:choose>
      <xsl:when test="node-name($template-node) = node-name($document-node)">
        <xsl:variable name="attrMatch">
          <xsl:for-each select="$template-node/@*[namespace-uri(.) = '']">
            <xsl:variable name="aname" select="local-name(.)"/>
            <xsl:variable name="attr" select="$document-node/@*[local-name(.) = $aname]"/>
            <xsl:choose>
              <xsl:when test="$attr = .">1</xsl:when>
              <xsl:otherwise>0</xsl:otherwise>
            </xsl:choose>
          </xsl:for-each>
        </xsl:variable>
        <xsl:choose>
          <xsl:when test="not(contains($attrMatch, '0'))">
            <xsl:value-of select="dbf:user-node-matches($template-node, $document-node)"/>
          </xsl:when>
          <xsl:otherwise>
            <xsl:value-of select="false()"/>
          </xsl:otherwise>
        </xsl:choose>
      </xsl:when>
      <xsl:otherwise>
        <xsl:value-of select="false()"/>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:function>

  <xsl:function name="dbf:node-matches" as="xs:boolean">
    <xsl:param name="template-node" as="element()"/>
    <xsl:param name="document-node" as="element()"/>

    <xsl:value-of
      select="
    (node-name($template-node) = node-name($document-node))
 and (
   every $i in $template-node/@*[namespace-uri(.) = '']
   satisfies
   for $j in $document-node/@*[local-name(.) = local-name($i)]
      return
      $i = $j)
  and 
  dbf:user-node-matches($template-node, $document-node)
 "
    />
  </xsl:function>

  <xsl:template match="/">
    <x>
      <xsl:variable name="t" as="element()">
        <xs:foo a="1" b="1" c="2"/>
      </xsl:variable>

      <xsl:variable name="d" as="element()">
        <xs:foo a="1" b="1" c="2"/>
      </xsl:variable>

      <xsl:copy-of select="($t,$d)"/>
      <m>
        <xsl:value-of select="string(dbf:node-matches1($t,$d))"/>
      </m>
      <m>
        <xsl:value-of select="string(dbf:node-matches($t,$d))"/>
      </m>

      <xsl:variable name="t" as="element()">
        <xs:foo a="1" b="1" c="2"/>
      </xsl:variable>
      <xsl:variable name="d" as="element()">
        <xs:foo a="2" b="1" c="2"/>
      </xsl:variable>

      <xsl:copy-of select="($t,$d)"/>
      <m>
        <xsl:value-of select="string(dbf:node-matches1($t,$d))"/>
      </m>
      <m>
        <xsl:value-of select="string(dbf:node-matches($t,$d))"/>
      </m>

      <xsl:variable name="t" as="element()">
        <xs:foo a="" b="1" c="2"/>
      </xsl:variable>
      <xsl:variable name="d" as="element()">
        <xs:foo b="1" c="2"/>
      </xsl:variable>

      <xsl:copy-of select="($t,$d)"/>
      <m>
        <xsl:value-of select="string(dbf:node-matches1($t,$d))"/>
      </m>
      <m>
        <xsl:value-of select="string(dbf:node-matches($t,$d))"/>
      </m>

      <xsl:variable name="t" as="element()">
        <xs:foo a="0" b="1" c="2"/>
      </xsl:variable>
      <xsl:variable name="d" as="element()">
        <xs:foo a="0" b="1" c="2" e="5"/>
      </xsl:variable>

      <xsl:copy-of select="($t,$d)"/>
      <m>
        <xsl:value-of select="string(dbf:node-matches1($t,$d))"/>
      </m>
      <m>
        <xsl:value-of select="string(dbf:node-matches($t,$d))"/>
      </m>

      <xsl:variable name="t" as="element()">
        <foo a="0" b="1" c="2"/>
      </xsl:variable>
      <xsl:variable name="d" as="element()">
        <xs:foo a="0" b="1" c="2" e="5"/>
      </xsl:variable>

      <xsl:copy-of select="($t,$d)"/>
      <m>
        <xsl:value-of select="string(dbf:node-matches1($t,$d))"/>
      </m>
      <m>
        <xsl:value-of select="string(dbf:node-matches($t,$d))"/>
      </m>
    </x>
  </xsl:template>

</xsl:stylesheet>
