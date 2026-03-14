<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test ordinal numbering in words -->
  <xsl:template match="doc">
    <xsl:variable name="n" select="1,2,3,10,11,13,20,21,78,100,115,134,200,201,210,230,1000,1005,1998,2000,2002,2138416"/>
    <xsl:variable name="w" select="'w'"/>
    <xsl:variable name="W" select="'W'"/>
    <xsl:variable name="Ww" select="'Ww'"/>
    <xsl:variable name="f" select="for $x in 1 to 10 return ($w, $W, $Ww)"/>
    <xsl:variable name="regex">
      <r>first</r>
      <r>SECOND</r>
      <r>Third</r>
      
      <r>tenth</r>
      <r>ELEVENTH</r>
      <r>Thirteenth</r>
      
      <r>twentieth</r>
      <r>TWENTY.*FIRST</r>
      <r>Seventy.*[Ee]ighth</r>
      
      <r>one.*hundredth</r>
      <r>ONE.*HUNDRED.*FIFTEENTH</r>
      <r>One.*Hundred.*Thirty.*[Ff]ourth</r>
      
      <r>two.*hundredth</r>
      <r>TWO.*HUNDRED.*FIRST</r>
      <r>Two.*Hundred.*Tenth</r>
      
      <r>two.*hundred.*thirtieth</r>
      <r>ONE.*THOUSANDTH</r>
      <r>One.*Thousand.*Fifth</r>
      
      <r>one.*thousand.*nine.*hundred.*ninety.*eighth</r>
      <r>TWO.*THOUSANDTH</r>
      <r>Two.*Thousand.*Second</r>
      
      <r>two.*million.*one.*hundred.*thirty.*eight.*four.*hundred.*sixteenth</r>                                 
    </xsl:variable>  
    <out>
      <xsl:for-each select="1 to count($n)">
        <xsl:variable name="words">
          <xsl:number value="$n[current()]" format="{$f[current()]}" ordinal="yes"/>
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
