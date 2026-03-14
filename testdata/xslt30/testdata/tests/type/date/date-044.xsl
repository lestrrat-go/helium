<?xml version="1.0" ?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs f"
xmlns:f="http://localhost/">

<?spec fo#date-time-values?>
  <!-- PURPOSE: XPath 2.0: test sorting on times. -->
  
  <xsl:variable name="x">
   <a>18:00:00</a>
   <a>18:00:00.002</a>
   <a>18:01:00Z</a>
   <a>19:00:30+01:00</a>
   <a>00:00:00</a>
   <a>00:00:00-01:00</a>
   <a>00:00:00+01:00</a>
  </xsl:variable>
  
  <xsl:param name="debug" required="no"/>

  <xsl:template match="/">
  <!-- rather than just display the sorted list, we save it and inspect it, to counter
       any dependencies on the implicit timezone -->
  <xsl:variable name="sorted" as="element(a)*">
    <xsl:for-each select="$x/a">
      <xsl:sort select="xs:time(.)" />
      <xsl:copy-of select="."/>
    </xsl:for-each>
  </xsl:variable>
  
  <xsl:if test="$debug"><xsl:message select="$sorted"/></xsl:if>
  
  <out>
      <xsl:for-each select="1 to count($sorted)-1">
        <xsl:variable name="i" select="." as="xs:integer"/>
        <xsl:choose>
        <xsl:when test="f:adjust($sorted[$i]) le f:adjust($sorted[$i+1])">
          <ok/>
        </xsl:when>
        <xsl:otherwise>
          <problem>at <xsl:value-of select="$i"/></problem>
        </xsl:otherwise>
        </xsl:choose>
      </xsl:for-each>
  </out>
  </xsl:template>
  
  <xsl:function name="f:adjust" as="xs:time">
    <xsl:param name="in" as="xs:string"/>
    <xsl:variable name="zoned" select="contains($in, '+') or contains($in, '-') or contains($in, 'Z')"/>
    <xsl:sequence select="if ($zoned) then xs:time($in) 
                          else adjust-time-to-timezone(xs:time($in), implicit-timezone())"/>
  </xsl:function>

</xsl:stylesheet>