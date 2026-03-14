<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:f = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs f"
  expand-text="yes">

  <!-- Cache=yes with a tail-recursive function.  -->
  <!-- From https://so.nwalsh.com/2020/10/09-fib -->
    
  <!-- You probably don't want to be mixing tail recursion and caching in this way.
       But this test aims to find out what happens if you do -->

<?spec xslt#stylesheet-functions?>

  <xsl:function name="f:fib"
    as="xs:integer">
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence
      select="f:tail-recursive-fib($n, 0, 1) * 1"/>
  </xsl:function>
  
  <xsl:function name="f:tail-recursive-fib"
    as="xs:integer"
    visibility="private"
    cache="yes">
    <xsl:param name="count" as="xs:integer"/>
    <xsl:param name="n" as="xs:integer"/>
    <xsl:param name="next" as="xs:integer"/>
    <xsl:choose>
      <xsl:when test="$count = 0">
        <xsl:sequence select="$n"/>
      </xsl:when>
      <xsl:otherwise>
        <xsl:sequence
          select="f:tail-recursive-fib(
          $count - 1, $next,
          $n + $next)"/>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:function>
  
  <xsl:template name="xsl:initial-template">
    <out>
      <pass1>{(30 to 40) ! f:fib(.)}</pass1>
      <pass2>{(30 to 40) ! f:fib(.)}</pass2>
    </out>
  </xsl:template>

</xsl:stylesheet>
