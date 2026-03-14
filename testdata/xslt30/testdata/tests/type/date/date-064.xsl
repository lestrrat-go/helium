<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" 
xmlns:z="http://z.com/" exclude-result-prefixes="xs z">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test English a.m / p.m formatting -->

    <xsl:param name="d1" as="xs:time" select="xs:time('00:00:00')"/>
    <xsl:param name="d2" as="xs:time" select="xs:time('00:05:00')"/>
    <xsl:param name="d3" as="xs:time" select="xs:time('01:00:00')"/>
    <xsl:param name="d4" as="xs:time" select="xs:time('09:30:00')"/>
    <xsl:param name="d5" as="xs:time" select="xs:time('12:00:00')"/>
    <xsl:param name="d6" as="xs:time" select="xs:time('15:15:00')"/>

  <xsl:function name="z:format">
    <xsl:param name="f"/>
    <xsl:for-each select="$d1, $d2, $d3, $d4, $d5, $d6">
       <xsl:sequence select="format-time(., $f, 'en', (), ())"/>
    </xsl:for-each>
  </xsl:function>
  
  <xsl:function name="z:norm">
    <!-- normalize results so A.M. and AM and A.m. and Am are all acceptable -->
    <xsl:param name="f"/>
    <xsl:sequence select="for $g in $f return replace($g, '\.?[mM]\.?', 'M')"/>
  </xsl:function> 

  <xsl:template match="/">
     <out>;
     <code x="{z:norm(z:format('[H01]:[m01] = [h][PNn,4-8]'))}"/>;
     <code x="{z:norm(z:format('[H01]:[m01] = [h][PNn,4-4]'))}"/>;  
     <code x="{z:format('[H01]:[m01] = [h][PNn,3-3]')}"/>;  
     <code x="{z:format('[H01]:[m01] = [h][PNn,2-2]')}"/>;  
     <code x="{z:format('[H01]:[m01] = [h][PNn,1-1]')}"/>;  
     <code x="{z:format('[H01]:[m01] = [h][PN,2-2]')}"/>;  
     <code x="{z:format('[H01]:[m01] = [h][Pn,2-2]')}"/>; 
     </out>     
  </xsl:template>

</xsl:stylesheet>