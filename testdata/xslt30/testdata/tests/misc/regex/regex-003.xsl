<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test replace() function. -->
  <?spec fo#func-replace?>

  <xsl:template match="/">
    <out>;
        <replace test="{replace('abba', 'a', 'A')}" expect="AbbA"/>;
        <replace test="{replace('abracadabra', 'a.*a', 'X')}" expext="X"/>;
        <replace test="{replace('abracadabra', 'a.*?a', 'X')}" expect="XcXbra"/>;
        <replace test="{replace('abracadabra', 'a([^b])', 'A$1')}" expect="abrAcAdabra"/>;
            
    </out>
  </xsl:template>

</xsl:stylesheet>