<?xml version="1.0"?><!--nspace.xsl-->
<!--XSLT 1.0 - http://www.CraneSoftwrights.com/training -->

<?spec xpath#axes?>
<!-- BUG: Saxon 5.3.2/018 -->
<!-- DESCRIPTION:
    In a location path using the namespace axis with a name test, for example namespace::x,
    all the namespace nodes are returned in the result, whether or not their name matches "x". -->
    
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                 version="2.0" xmlns:a="http://a.com/" xmlns:b="http://b.com/">
<xsl:output method="text"/>

<xsl:template match="/">                         <!--root rule-->
   <xsl:for-each select="//*[local-name(.)='hello']">
//*[name(.)='hello'] '<xsl:value-of select="."/>'
   </xsl:for-each>
   <xsl:for-each select="//hello">
//hello '<xsl:value-of select="."/>'
   </xsl:for-each>
   <xsl:for-each select="//a:hello">
//a:hello '<xsl:value-of select="."/>'
   </xsl:for-each>
   <xsl:for-each select="//b:hello">
//b:hello '<xsl:value-of select="."/>'
   </xsl:for-each>
   <xsl:for-each select="(//hello)[1]">
   <xsl:for-each select="namespace::*[local-name(.)='a' or name(.)='b']"><xsl:sort select="name()"/>
namespace::*[local-name(.)='a' or name(.)='b'] '<xsl:value-of select="."/>'
   </xsl:for-each>
   <xsl:for-each select="@a">
@a '<xsl:value-of select="."/>'
   </xsl:for-each>
   <xsl:for-each select="@a:a">
@a:a '<xsl:value-of select="."/>'
   </xsl:for-each>
   <xsl:for-each select="@b:a">
@b:a '<xsl:value-of select="."/>'
   </xsl:for-each>
   <xsl:for-each select="namespace::*"><xsl:sort select="name()"/>
namespace::* '<xsl:value-of select="name(.)"/>' '<xsl:value-of select="."/>'
   </xsl:for-each>
   <xsl:for-each select="namespace::b">
namespace::b '<xsl:value-of select="name(.)"/>' '<xsl:value-of select="."/>'
   </xsl:for-each>
   <xsl:for-each select="namespace::b:a">
namespace::b:a '<xsl:value-of select="name(.)"/>' '<xsl:value-of select="."/>'
   </xsl:for-each>
   </xsl:for-each>
</xsl:template>

</xsl:stylesheet>

