<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- Purpose: Composite key with variable number of components -->
  
  <xsl:variable name="poem">
    <verse>January brings the snow,
    Makes our feet and fingers glow.</verse>
    
    <verse>February brings the rain,
    Thaws the frozen lake again.</verse>
    
    <verse>March brings breezes loud and shrill,
    Stirs the dancing daffodil.</verse>
    
    <verse>April brings the primrose sweet,
    Scatters daises at our feet.</verse>
    
    <verse>May brings flocks of pretty lambs,
    Skipping by their fleecy damns.</verse>
    
    <verse>June brings tulips, lilies, roses,
    Fills the children's hand with posies.</verse>
    
    <verse>Hot July brings cooling showers,
    Apricots and gillyflowers.</verse>
    
    <verse>August brings the sheaves of corn,
    Then the harvest home is borne.</verse>
    
    <verse>Warm September brings the fruit,
    Sportsmen then begin to shoot.</verse>
    
    <verse>Fresh October brings the pheasants,
    Then to gather nuts is pleasant.</verse>
    
    <verse>Dull November brings the blast,
    Then the leaves are whirling fast.</verse>
    
    <verse>Chill December brings the sleet,
    Blazing fire, and Christmas treat. </verse>
  </xsl:variable>
  
  <xsl:key name="k" match="verse" use="substring-before(., ',')!tokenize(.)" composite="1"/> 
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:value-of select="key('k',tokenize('Hot July brings cooling showers'), $poem) => normalize-space()"/>      
    </out>
  </xsl:template>

</xsl:stylesheet>
