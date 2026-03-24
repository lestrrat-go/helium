<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://example.com/functions"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <!-- Saxon bug 5728 -->
  
  <xsl:template name="xsl:initial-template">
    <xsl:variable name="dms" select="'spoon'"/>
    <doc>
      <inline>
        <xsl:try>
          <xsl:variable name="part"
            select="tokenize($dms, '[\s°''&quot;]+') ! xs:double(.)"/>
          <xsl:sequence select="(abs($part[1]) + ($part[2] div 60.0) + ($part[3] div 3600.0))
            * (if ($part[1] lt 0) then -1 else +1)"/>
          <xsl:catch errors="*">
            <xsl:message select="'Failed to parse GPS DMS: ' || $dms"/>
            <xsl:sequence select="'NOPE'"/>
          </xsl:catch>
        </xsl:try>
      </inline>
      <function>
        <xsl:sequence select="f:gps($dms)"/>
      </function>
      <message>
        <xsl:variable name="x" select="f:gps($dms)"/>
        <xsl:message select="'MSG', $x"/>
      </message>
      <nested-functions>
        <xsl:sequence select="f:reverse-geocode($dms)"/>
      </nested-functions>
    </doc>
  </xsl:template>
  
  <xsl:function name="f:reverse-geocode" as="xs:string?">
    <xsl:param name="dms" as="xs:string?"/>
    <xsl:variable name="x" select="f:gps($dms)"/>
    <xsl:message select="'NESTED F', $x"/>
    <xsl:sequence select="string(f:gps($dms))"/>
  </xsl:function>
  
  <xsl:function name="f:gps" as="xs:double?">
    <xsl:param name="dms" as="xs:string?"/>
    <xsl:if test="exists($dms)">
      <!-- expecting something like 51° 36' 48.07 -->
      <xsl:try>
        <xsl:variable name="part"
          select="tokenize($dms, '[\s°''&quot;]+') ! xs:double(.)"/>
        <xsl:sequence select="(abs($part[1]) + ($part[2] div 60.0) + ($part[3] div 3600.0))
          * (if ($part[1] lt 0) then -1 else +1)"/>
        <xsl:catch errors="*">
          <xsl:message select="'Failed to parse GPS DMS: ' || $dms"/>
          <xsl:sequence select="()"/>
        </xsl:catch>
      </xsl:try>
    </xsl:if>
  </xsl:function>
  
</xsl:stylesheet>