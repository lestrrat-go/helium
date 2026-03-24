<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://mhk.me.uk/my"
                exclude-result-prefixes="my xs"
		version="3.0">

<!-- Tests use of xsl:evaluate() within a predicate -->
<!-- Bug 6.1/017 -->
<!-- Formerly test saxon056 -->

<!-- RUN as: ./saxon test.xml test.xsl sortby1=@category -->
<!-- test.xml can be empty file -->


<!-- primary sort fields -->

<!-- ******************************************************************************************************** -->
<!-- **************** change field below from "param" to "variable" and it works as intended on version 5.5.1 ************** -->
<!-- ******************************************************************************************************** -->

<xsl:param name="sortby1" select="'@category'"/>

<xsl:variable name="indata">
 <things>
  <thing category="forbin" name="SVC board vhdl code consolidation"/>
  <thing category="forbin" name="IOgen template incorporation"/>
  <thing category="notforbin" name="General m4-template maintenance"/>
 </things>
</xsl:variable>

<!-- main -->
<xsl:template name="main">
 <xsl:variable name="workaround" select="$indata/things/thing[my:evaluate(.,$sortby1)='forbin']"
                />
 <out>               
 <xsl:for-each select="$workaround">
  <td><xsl:value-of select="@name"/></td>
 </xsl:for-each>
 </out>  
</xsl:template>

<xsl:function name="my:evaluate">
  <xsl:param name="contextItem" as="item()"/>
  <xsl:param name="xpath" as="xs:string"/>
  <xsl:evaluate xpath="$xpath" context-item="$contextItem"/>
</xsl:function>

</xsl:stylesheet>

