<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test numbering in words -->
  <xsl:template match="doc">
    <xsl:variable name="n" select="3,10,13,20,100,115,134,200,201,210,230,1000,1005,2134816"/>
    <xsl:variable name="w" select="'w'"/>
    <xsl:variable name="W" select="'W'"/>
    <xsl:variable name="Ww" select="'Ww'"/>
    <xsl:variable name="f" select="$w,$w,$w,$w,$W,$W,$W,$w,$w,$w,$w,$W,$W,$Ww"/>
    <xsl:variable name="regex">
      <r>three</r>
      <r>ten</r>
      <r>thirteen</r>
      <r>twenty</r>
      <r>ONE.*HUNDRED</r>
      <r>ONE.*HUNDRED.*FIFTEEN</r>
      <r>ONE.*HUNDRED.*THIRTY.*FOUR</r>
      <r>two.*hundred</r>
      <r>two.*hundred.*one</r>
      <r>two.*hundred.*ten</r>
      <r>two.*hundred.*thirty</r>
      <r>ONE.*THOUSAND</r>
      <r>ONE.*THOUSAND.*FIVE</r>
      <r>Two.*Million.*One.*Hundred.*Thirty.*[Ff]our.*Thousand.*Eight.*Hundred.*Sixteen</r>
    </xsl:variable>  
    <out>
      <xsl:for-each select="1 to count($n)">
        <xsl:variable name="words">
          <xsl:number value="$n[current()]" format="{$f[current()]}"/>
        </xsl:variable>  
        <z>
          <xsl:choose>
            <xsl:when test="matches($words, $regex/r[current()])">OK</xsl:when>
            <xsl:otherwise><xsl:value-of select="$words"/></xsl:otherwise>
          </xsl:choose>
        </z>  
      </xsl:for-each>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
