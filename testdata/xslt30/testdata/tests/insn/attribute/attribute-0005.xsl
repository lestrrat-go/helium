<?xml version="1.0" encoding="UTF-8"?>

  <!-- FileName: atrs65 -->
    <!-- Purpose: Test that the value of the name attribute is cast to a string if supplied as a boolean
       (very artificial case, and results have no canonical form, but legal XSLT). -->

  
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:p="http://ns.p.com/"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="p xs">
  
  <xsl:param name="ns" select="false()"/>

  <xsl:template name="main">
    <xsl:variable name="one">
      <out>
        <xsl:attribute name="p:local" namespace="{ $ns }">content</xsl:attribute>
      </out>
    </xsl:variable>
    <xsl:variable name="two">
      <out p:local="content" xmlns:p="false"/>
    </xsl:variable>
    <!-- compare with expected results using deep-equal to avoid canonicalization problems in test driver -->
    <result value="{deep-equal($one, $two)}"/>  
  </xsl:template>


</xsl:stylesheet>