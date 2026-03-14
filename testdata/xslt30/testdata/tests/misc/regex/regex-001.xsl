<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test matches() function with two arguments. -->
  <?spec fo#func-matches?>

  <xsl:template match="/">
    <out>;
      <true result="{matches('abracadabra', 'bra')}"/>;
      <true result="{matches('abracadabra', '.*')}"/>;
      <true result="{matches('abracadabra', 'a.*a')}"/>;
      <true result="{matches('abracadabra', 'bra$')}"/>;   
      <false result="{matches('abracadabra', 'arab')}"/>;
      <false result="{matches('abracadabra', '^bra')}"/>;      
    </out>
  </xsl:template>

</xsl:stylesheet>