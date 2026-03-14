<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:local = "http://local.com/" exclude-result-prefixes="local">

<!-- test treatment of whitespace after calling a function -->

  <xsl:template match="/">
      <out>;
        <xsl:value-of select="'[', local:test(3), ']'"/>;                      
      </out>
  </xsl:template>

<xsl:function name="local:test">
   <xsl:param name="p1"/>
</xsl:function>


</xsl:stylesheet>
