<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:math="http://www.w3.org/2005/xpath-functions/math"
  xmlns:xd="http://www.oxygenxml.com/ns/doc/xsl" xmlns:lc="http://xoev.de/latinchars"
  xpath-default-namespace="http://xoev.de/latinchars" exclude-result-prefixes="xs math xd"
  version="3.0">
  <!-- Saxon issue 4446 -->

  <xsl:import-schema namespace="http://xoev.de/latinchars" schema-location="lc-simple.xsd"/>
  <xsl:output method="xml" indent="yes"/>

  <!-- The input is validated against lc-simple.xsd 
       It seems to be important, that this is a global variable.
       
       When this variable is locally defined in the initial template,
       $nfd-base-2 still has only one element (which is wrong), 
       but this one element has a parent element.
       
       But when this variable is global, then the one element has no parent element at all. -->
  <xsl:variable name="characterSet" as="document-node(schema-element(lc:characterSet))">
    <xsl:document validation="strict">
      <xsl:sequence select="/characterSet"/>
    </xsl:document>
  </xsl:variable>


  <xsl:template match="/">
    <!-- three times a sequence of base elements. they shoud be identical, but are not -->
    <xsl:variable name="nfd-base-1" as="element(base)*" select="$characterSet//extension/nfd/base"/>
    <xsl:variable name="nfd-base-2" as="element(base)*" select="$characterSet//nfd/base"/>
    <xsl:variable name="nfd-base-3" as="element(base)*" select="$characterSet//*/base"/>
    <xsl:variable name="nfd-base-4" as="element(base)*" select="$characterSet//base"/>
    <bug-4446 xmlns="http://xoev.de/latinchars">
      
      <sequence remark="nfd-base-1" selection='$characterSet//extension/nfd/base'>
        <xsl:attribute name="number-of-parents" select="count($nfd-base-1/parent::*)"/>
        <xsl:apply-templates select="$nfd-base-1"/>
      </sequence>
      
      <sequence remark="nfd-base-2" selection='$characterSet//nfd/base'>
        <xsl:attribute name="number-of-parents" select="count($nfd-base-2/parent::*)"/>
        <xsl:apply-templates select="$nfd-base-2"/>
      </sequence>
      
      <sequence remark="nfd-base-3" selection='$characterSet//*/base'>
        <xsl:attribute name="number-of-parents" select="count($nfd-base-3/parent::*)"/>
        <xsl:apply-templates select="$nfd-base-3"/>
      </sequence>
      
      <sequence remark="nfd-base-4" selection='$characterSet//base'>
        <xsl:attribute name="number-of-parents" select="count($nfd-base-4/parent::*)"/>
        <xsl:apply-templates select="$nfd-base-4"/>
      </sequence>

    </bug-4446>
  </xsl:template>
  

  <xsl:template match="base">
    <xsl:copy copy-namespaces="no">
      <xsl:attribute name="exists-parent-element" select="exists(parent::*)"/>
      <xsl:copy-of select="text()"/>
    </xsl:copy>
  </xsl:template>

</xsl:stylesheet>
