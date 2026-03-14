<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <!-- See Saxon bug 2032460. Bug in lazy evaluation of a varaible whose contents
       are constructed using xsl:for-each-group -->


  <xsl:template match="/">
    <xsl:variable name="records" as="element()+">
      <record view-id="2009072185">
        <field/>
      </record>
      <record view-id="2009072185">
        <field/>
      </record>
    </xsl:variable>

    <xsl:variable name="source" as="element()+">
      <xsl:for-each-group select="$records" group-by="xs:string(@view-id)">
        <xsl:sequence select="."/>
      </xsl:for-each-group>
    </xsl:variable>

    <!--<xsl:message select="$source"/>-->

    <xsl:variable name="alias" as="xs:string+">
      <xsl:for-each select="$source">
        <xsl:choose>
          <xsl:when test="last() = 1">
            <xsl:sequence select="'T'"/>
          </xsl:when>
          <xsl:otherwise>
            <xsl:sequence select="concat('T', position())"/>
          </xsl:otherwise>
        </xsl:choose>
      </xsl:for-each>
    </xsl:variable>

    <select>
      <xsl:for-each select="$source">
        <field alias="{$alias}"/>
      </xsl:for-each>

      <xsl:variable name="index" as="xs:integer" select="
        for 
          $i in 1 to count($source),
          $source-record in $source[$i][. is $source[1]]
        return
          $i"/>

      <field alias="{$alias[$index]}"/>
    </select>
  </xsl:template>

</xsl:stylesheet>
