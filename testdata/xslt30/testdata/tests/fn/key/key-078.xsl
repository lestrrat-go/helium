<?xml version="1.0" encoding="ISO-8859-1"?> 
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                              xmlns:tt="http://www.powerware.com/nz/tt">
  <!-- test two key definitions with same name matching overlapping nodesets -->                            
  <xsl:key name="tt:testKey" match="node()" use="('a', '*b*')"/>
  <xsl:key name="tt:testKey" match="node()" use="('a', 'b')"/>
  <xsl:template match="/">
    <out a="{count(key('tt:testKey', '*b*'))}" b="{count(key('tt:testKey', 'b'))}"/>
  </xsl:template>
</xsl:stylesheet>
