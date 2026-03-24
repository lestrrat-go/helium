<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs x">

<!-- deterministic function: check that it handles node identity correctly  -->

<?spec xslt#stylesheet-functions?>

<xsl:function name="x:test" new-each-time="no" as="element()">
  <xsl:param name="n" as="element()"/>
  <b><xsl:copy-of select="$n"/></b>
</xsl:function>

  <xsl:template match="/">
    <xsl:variable name="in">
      <xsl:for-each select="1 to 100">
        <a/>
      </xsl:for-each>
    </xsl:variable>
    <xsl:variable name="temp" as="element()*">
      <xsl:for-each select="$in/a">
        <xsl:sequence select="x:test(.)"/>
      </xsl:for-each>
    </xsl:variable>  
    <out><xsl:value-of select="count($temp/.)"/></out>   
  </xsl:template>


</xsl:stylesheet>
