<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0" 
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:str="java:java.lang.String"
 xmlns:my="http://ohmy/">
 
 <!-- Test function-available() with two arguments -->
 <?category function-available?>
 
 <xsl:param name="one" select="1"/>
 <xsl:param name="two" select="2"/>
 <xsl:param name="three" select="3"/>  
 
 
  
   <xsl:template match="/">
	<out  a="{function-available('true', 0)}"
      	b="{function-available('translate', $three)}"
      	c="{function-available('function-available', 1)}"
      	d="{function-available('function-available', $two)}"
      	e="{function-available('my:f')}"
      	f="{function-available('my:f', $two)}"
      	g="{function-available('concat', 17)}" 
      	h="{function-available('true', 1)}" 
      	i="{function-available('translate', $two)}"
      	j="{function-available('function-available', 3)}" 
      	k="{function-available('function-available', $three)}"
      	l="{function-available('my:f', 1)}" 
      	m="{function-available('concat', 0)}"/>		
	
   </xsl:template>
   
   <xsl:function name="my:f">
     <xsl:param name="i"/>
     <xsl:param name="j"/>
     <xsl:value-of select="$i+$j"/>
   </xsl:function>

</xsl:stylesheet>

