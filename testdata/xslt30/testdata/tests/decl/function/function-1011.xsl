<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0" 
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 xmlns:my="http://my.com/"
 exclude-result-prefixes="my xs"
>
 
 <!-- Test whether function evaluation is pipelined. This test will return -100 if 
      evaluation is pipelined, or throw a divide-by-zero error if not -->
 
 <xsl:param name="start" select="-1" as="xs:integer"/>
 
   <xsl:template match="/">
     <out result="{my:f($start)[1]}"/>
   </xsl:template>
   
   <xsl:function name="my:f" as="xs:double*">
     <xsl:param name="start"/>
     <xsl:sequence select="for $i in $start to $start+10 return 100 div $i"/>
   </xsl:function>

</xsl:stylesheet>

