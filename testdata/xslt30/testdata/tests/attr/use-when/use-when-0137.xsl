<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">


  <!-- PURPOSE: precednece of static variables when the same module is imported more than once. -->
  
  <!-- See bug 24478 -->
   
  <xsl:import href="use-when-0137a.xsl"/>
  
  <xsl:variable name="one" static="yes" select="$flip-flop"/> <!--should be true -->
  
  <xsl:import href="use-when-0137b.xsl"/>
  
  <xsl:variable name="two" static="yes" select="$flip-flop"/> <!--should be false -->
  
  <xsl:import href="use-when-0137a.xsl"/>
  
  <xsl:variable name="three" static="yes" select="$flip-flop"/> <!--should be true -->
    
  <xsl:template name="main" use-when="$one and not($two) and $three">
    <ok/>
  </xsl:template>

</xsl:stylesheet>