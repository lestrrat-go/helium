<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                >

<!-- PURPOSE: Test namespace nodes -->

<xsl:template match="/">
<out>
<copy-of-test>
<xsl:copy-of select="."/>
</copy-of-test>
<copy-test>
   <xsl:for-each select="/*/*">
       <xsl:copy>CC</xsl:copy>
   </xsl:for-each>
</copy-test>
<copy-test>
   <xsl:for-each select="/*/*/*">
       <xsl:copy>PP</xsl:copy>
   </xsl:for-each>
</copy-test>
<lre-test>
    <doc xmlns:z="http://z.uri/" xmlns="http://default.uri/">
      <chap xmlns:z="http://new-z.uri/" xmlns="http://chap-default.uri/">
         <para xmlns=""/>
      </chap>
    </doc>
</lre-test>
<mask-test>
<xsl:for-each select="//c:chap/namespace::*" xmlns:c="http://chap-default.uri/">
    <xsl:sort select="name()"/>
  <xsl:value-of select="name()"/> = <xsl:value-of select="."/>;
</xsl:for-each>
</mask-test>
<mask-test>
<xsl:for-each select="//para/namespace::node()" >
    <xsl:sort select="name()"/>
  <xsl:value-of select="name()"/> = <xsl:value-of select="."/>;
</xsl:for-each>
</mask-test>
</out>
</xsl:template>
 
</xsl:stylesheet>
