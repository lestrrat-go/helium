<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" 
exclude-result-prefixes="xs"
version="3.0">

  <!-- Purpose: A key that mixes several data types -->

  <xsl:output method="xml" encoding="UTF-8" indent="no"/>

  <xsl:key name="k" match="town" use="@state"/>
  <xsl:key name="k" match="town[@state='NH']" use="string-length(@name)"/>
  <xsl:key name="k" match="town[@state='RI']" use="xs:untypedAtomic(string-length(@name))"/>
  <xsl:key name="k" match="town[@state='VT']" use="ends-with(@name, 'field')"/>

  <xsl:template match="/">
    <out>
      <fields>
        <xsl:value-of select="key('k', true())/concat(@name, @state)"/>
      </fields> 
      <non-fields>
        <xsl:value-of select="key('k', false())/concat(@name, @state)"/>
      </non-fields> 
      <sevens>
        <xsl:value-of select="key('k', 7)/concat(@name, @state)"/>
      </sevens>
      <u-sevens>
        <xsl:value-of select="key('k', '7')/concat(@name, @state)"/>
      </u-sevens>
      <nh>
        <xsl:value-of select="key('k', 'NH')/concat(@name, @state)"/>
      </nh>  
    </out>
  </xsl:template>

</xsl:stylesheet>
