<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="3.0" 
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:fn="http://www.w3.org/2005/xpath-functions"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 xmlns:my="http://ohmy/"
 exclude-result-prefixes="fn my xs">
 
 <!-- Test function-available() with EQName arguments -->
 <?category function-available?>
 
 <xsl:param name="one" select="1"/>
 <xsl:param name="two" select="2"/>
 <xsl:param name="three" select="3"/> 
 
 <xsl:param name="integer">Q{http://www.w3.org/2001/XMLSchema}integer</xsl:param>
 <xsl:param name="abs">Q{http://www.w3.org/2005/xpath-functions}abs</xsl:param>
 <xsl:param name="wrong">Q{}abs</xsl:param>
  
 
 
  
   <xsl:template match="/">
	<out> 
            <a><xsl:value-of select="function-available($integer, $one)"/></a>
            <b><xsl:value-of select="function-available($abs, $one)"/></b>
            <c><xsl:value-of select="function-available($wrong, 1)"/></c>
            <d><xsl:value-of select="function-available('Q{http://ohmy/}f', $two)"/></d>
	</out>
   </xsl:template>
   
   <xsl:function name="my:f">
     <xsl:param name="i"/>
     <xsl:param name="j"/>
     <xsl:value-of select="$i+$j"/>
   </xsl:function>

</xsl:stylesheet>

