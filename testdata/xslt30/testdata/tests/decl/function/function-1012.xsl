<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0" 
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 xmlns:my="http://my.com/"
 exclude-result-prefixes="my xs"
>
 
 <!-- Test numeric promotion of function result -->
 
 <xsl:param name="start" select="5" as="xs:integer"/>
 
   <xsl:template match="/">
     <out result="{my:f($start)}"/>
   </xsl:template>
   
   <xsl:function name="my:f" as="xs:double*">
     <xsl:param name="start"/>
     <xsl:sequence select="$start + 0.5"/>
   </xsl:function>

</xsl:stylesheet>

