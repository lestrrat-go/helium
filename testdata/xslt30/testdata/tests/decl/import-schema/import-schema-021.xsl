<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:hat="http://hats.uri/"
  exclude-result-prefixes="xs hat">

  <!-- PURPOSE: Test use of castable with user-defined types. -->


  <xsl:import-schema namespace="http://hats.uri/" schema-location="schema030.xsd"/>

  <xsl:template match="/">
    <out>
      <hatsize>
        <t>
          <xsl:value-of select="4 castable as hat:hatsize"/>
        </t>
        <t>
          <xsl:value-of select="7 castable as hat:hatsize"/>
        </t>
        <t>
          <xsl:value-of select="10 castable as hat:hatsize"/>
        </t>
        <f>
          <xsl:value-of select="3 castable as hat:hatsize"/>
        </f>
        <f>
          <xsl:value-of select="11 castable as hat:hatsize"/>
        </f>
      </hatsize>
      <bighatsize>
        <f>
          <xsl:value-of select="4 castable as hat:bighatsize"/>
        </f>
        <t>
          <xsl:value-of select="7 castable as hat:bighatsize"/>
        </t>
        <t>
          <xsl:value-of select="10 castable as hat:bighatsize"/>
        </t>
        <f>
          <xsl:value-of select="3 castable as hat:bighatsize"/>
        </f>
        <f>
          <xsl:value-of select="11 castable as hat:bighatsize"/>
        </f>
      </bighatsize>
      <shoesize>
        <t>
          <xsl:value-of select="-7 castable as hat:shoesize"/>
        </t>
        <t>
          <xsl:value-of select="-5 castable as hat:shoesize"/>
        </t>
        <t>
          <xsl:value-of select="-3 castable as hat:shoesize"/>
        </t>
        <f>
          <xsl:value-of select="-8 castable as hat:shoesize"/>
        </f>
        <f>
          <xsl:value-of select="-2 castable as hat:shoesize"/>
        </f>
      </shoesize>
    </out>
  </xsl:template>

</xsl:stylesheet>
