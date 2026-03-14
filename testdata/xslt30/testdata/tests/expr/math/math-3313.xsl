<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:f="http://f.com/" exclude-result-prefixes="f xs" xmlns:xs="http://www.w3.org/2001/XMLSchema">

 <!-- test decimal mod -->

 <xsl:template match="/">
  <out>
   <aa ans="{1.1 mod 1.2}"/>
   <ab ans="{5.2 mod 2.0}"/>
   <ac ans="{5.2 mod -2.0}"/>
   <ad ans="{-5.2 mod 2.0}"/>
   <ae ans="{-5.2 mod -2.0}"/>
   <af ans="{5.2 mod 2.5}"/>
   <ag ans="{5.2 mod -2.5}"/>
   <ah ans="{-5.2 mod 2.5}"/>
   <ai ans="{-5.2 mod -2.5}"/>
   <aj ans="{1.2 mod 1.1}"/>
   <ak ans="{2.4 mod 1.2}"/>
   <al ans="{0.0 mod 123.4}"/>
   <am ans="{10000000000000000000000000000000000 mod 0.1}"/>
   <an ans="{19999999999999999999999999999999999 mod 0.333333333333333333333333333333}"/>
  </out>
 </xsl:template>
</xsl:stylesheet>
