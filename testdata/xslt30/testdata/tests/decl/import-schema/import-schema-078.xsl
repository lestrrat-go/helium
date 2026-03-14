<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        exclude-result-prefixes="xs"
        default-validation="preserve">

<!-- PURPOSE: Test that elements of type xs:ID are recognized -->

<?spec fo#func-id?>




  <xsl:template match="/">
    <xsl:variable name="n">
    <out>
      <person>
        <code xsl:type="xs:ID">asdf</code>
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </person>
      <person>
        <code xsl:type="xs:ID">ghjk</code>
        <first>Penny</first>
        <surname>Whelan</surname>
      </person>
      <person>
      <code xsl:type="xs:ID">poiu</code>
        <last>Kay</last>
        <first>Philippa</first>
        <middle>H</middle>
      </person> 
    </out>
    </xsl:variable>
    <ans>
      <xsl:copy-of select="$n/id('ghjk')/.."/>               
    </ans>       
  </xsl:template>
  
</xsl:stylesheet>