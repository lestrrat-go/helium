<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:f="http://f.com/" exclude-result-prefixes="f xs" xmlns:xs="http://www.w3.org/2001/XMLSchema">

 <!-- test idiv on different operand types -->

 <xsl:template match="/">
  <out>
   <aa ans="{10.1 idiv 3}"/>
   <ab ans="{-3 idiv 10}"/>
   <ac ans="{5.2 idiv -2.0}"/>
   <ad ans="{-5.2 idiv 2.0}"/>
   <ae ans="{-5.2 idiv -2.0}"/>
   <af ans="{5.2 idiv 2.0}"/>
   <ac ans="{5.2e0 idiv -2.0}"/>
   <ad ans="{-5.2e0 idiv 2.0}"/>
   <ae ans="{-5.2e0 idiv -2.0}"/>
   <af ans="{5.2e0 idiv 2.0}"/>
   <ac ans="{xs:float(5.2e0) idiv -2.0}"/>
   <ad ans="{xs:float(-5.2e0) idiv 2.0}"/>
   <ae ans="{xs:float(-5.2e0) idiv -2.0}"/>
   <af ans="{xs:float(5.2e0) idiv 2.0}"/>
  </out>
 </xsl:template>
</xsl:stylesheet>
