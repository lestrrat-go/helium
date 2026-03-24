<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0" 
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:fn="http://www.w3.org/2005/xpath-functions"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 xmlns:my="http://ohmy/"
 exclude-result-prefixes="fn my xs">
 
 <!-- Test function-available() with dynamic arguments -->
 <?category function-available?>
 
 <xsl:param name="one" select="1"/>
 <xsl:param name="two" select="2"/>
 <xsl:param name="three" select="3"/> 
 
 <xsl:param name="true">true</xsl:param>
 <xsl:param name="translate">fn:translate</xsl:param>
 <xsl:param name="function-available">function-available</xsl:param>
 <xsl:param name="myf">my:f</xsl:param>
 <xsl:param name="xsstring">xs:string</xsl:param>
  
 
 
  
   <xsl:template match="/">
	<out>; 
            <a1><xsl:value-of select="function-available($true, 0)"/> should be true</a1>;
            <a1><xsl:value-of select="function-available($translate, $three)"/> should be true</a1>;
            <a1><xsl:value-of select="function-available($function-available, 1)"/> should be true</a1>;
            <a1><xsl:value-of select="function-available($function-available, $two)"/> should be true</a1>;
            <a1><xsl:value-of select="function-available($myf)"/> should be true</a1>;
            <a1><xsl:value-of select="function-available($myf, $two)"/> should be true</a1>;
            <a1><xsl:value-of select="function-available($xsstring)"/> should be true</a1>;
            <a1><xsl:value-of select="function-available($xsstring, $one)"/> should be true</a1>;

            
            <a1><xsl:value-of select="function-available($true, 1)"/> should be false</a1>;
            <a1><xsl:value-of select="function-available($translate, 4)"/> should be false</a1>;
            <a1><xsl:value-of select="function-available($function-available, 5)"/> should be false</a1>;
            <a1><xsl:value-of select="function-available($function-available, $three)"/> should be false</a1>;
            <a1><xsl:value-of select="function-available($myf, $one)"/> should be false</a1>;
            <a1><xsl:value-of select="function-available($xsstring, $two)"/> should be false</a1>;      
	</out>
   </xsl:template>
   
   <xsl:function name="my:f">
     <xsl:param name="i"/>
     <xsl:param name="j"/>
     <xsl:value-of select="$i+$j"/>
   </xsl:function>

</xsl:stylesheet>

