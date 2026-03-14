<?xml version="1.0"?> 
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Exercise potential optimization when position() appears in a predicate. -->

  <xsl:output method="xml" indent="no" encoding="UTF-8"/>

  <xsl:template match="doc">
    <out>
      <xsl:for-each-group select="*" group-starting-with="*[position() mod 2 = 1]">
         <group>
           <xsl:copy-of select="current-group()"/>
         </group>
      </xsl:for-each-group>
    </out>
  </xsl:template>

</xsl:stylesheet>
