<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0" 
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:my="http://foobar/"
 exclude-result-prefixes="my">
 
 <!-- Test that a stylesheet function can return zero-length text nodes -->
  
   <xsl:template match="/">
	<out count="{count(my:f())}">
	  <xsl:copy-of select="my:f()"/>
	</out>
   </xsl:template>
   
   <xsl:function name="my:f">
     <xsl:text/>
     <xsl:value-of select="''"/>
     <xsl:text/>
   </xsl:function>

</xsl:stylesheet>

