<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?><xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:x = "http://xxx.com/"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="xs x">

<!-- elidable function: tempt the optimizer with an opportunity for loop-lifting  -->

<?spec xslt#stylesheet-functions?>

<xsl:function name="x:test" new-each-time="maybe" as="element()">
  <xsl:message>With a smart optimizer, this message will only appear once.</xsl:message>
  <x/>
</xsl:function>

  <xsl:template match="/">
    <xsl:variable name="temp" as="element()*">
      <xsl:for-each select="1 to 100">
        <xsl:sequence select="x:test()"/>
      </xsl:for-each>
    </xsl:variable>  
    <out total="{count($temp)}" distinct="{count($temp/.)}"/>
  </xsl:template>


</xsl:stylesheet>
