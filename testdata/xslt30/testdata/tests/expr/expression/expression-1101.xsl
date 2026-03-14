<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test document()/key() -->
<?spec xslt#keys?>

<xsl:key name="city" match="city" use="@country"/>
<xsl:variable name="cities">
    <city name="Paris" country="France"/>
    <city name="Madrid" country="Spain"/>
    <city name="Vienna" country="Austria"/>
    <city name="Barcelona" country="Spain"/>
    <city name="Salzburg" country="Austria"/>
    <city name="Bonn" country="Germany"/>
    <city name="Lyon" country="France"/>
    <city name="Hannover" country="Germany"/>
    <city name="Calais" country="France"/>
    <city name="Berlin" country="Germany"/>
</xsl:variable>


<xsl:template match="/">
  <out>
    <xsl:for-each select="document('')/key('city','France')">
       <xsl:value-of select="@name"/>;
    </xsl:for-each>
  </out>
</xsl:template>
 
</xsl:stylesheet>
