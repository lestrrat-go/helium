<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if two
                     xsl:output declarations within an output definition specify explicit values
                  for the same attribute (other than cdata-section-elements and
                     use-character-maps), with the values of the attributes being not
                  equal, unless there is another xsl:output declaration within the
                  same output definition that has
                  higher import precedence and that specifies an explicit value for the same
                  attribute. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1560?>

   <xsl:output name="a" method="xml"/>
   <xsl:output name="a" method="html"/>

  <xsl:template name="main">
      <out>
         <e/>
      </out>
  </xsl:template>
  




</xsl:stylesheet>
