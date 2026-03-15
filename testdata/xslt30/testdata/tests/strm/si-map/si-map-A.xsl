<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
  <xsl:import-schema schema-location="../docs/loans.xsd"/>  
   
  <!-- within xsl:source-document, use xsl:map -->
  
  <xsl:template name="m-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM">
          <xsl:variable name="m" as="map(*)">
            <xsl:map>
              <xsl:map-entry key="'author'" select="string(AUTHOR)"/>
              <xsl:map-entry key="'title'" select="string(TITLE)"/>
              <xsl:map-entry key="'price'" select="number(PRICE)"/>
              <xsl:map-entry key="'publisher'" select="string(PUBLISHER)"/>
            </xsl:map>
          </xsl:variable>
          <book author="{$m('author')}" title="{$m('title')}" price="{$m('price')}" publisher="{$m('publisher')}"/>     
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:map with multi-valued map entries -->
  
  <xsl:template name="m-002" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
          <xsl:variable name="m" as="map(*)">
            <xsl:map>
              <xsl:map-entry key="'authors'" select="/BOOKLIST/BOOKS/ITEM/AUTHOR/string()"/>
              <xsl:map-entry key="'titles'" select="/BOOKLIST/BOOKS/ITEM/TITLE/string()"/>
              <xsl:map-entry key="'prices'" select="/BOOKLIST/BOOKS/ITEM/PRICE/number()"/>
              <xsl:map-entry key="'publishers'" select="/BOOKLIST/BOOKS/ITEM/PUBLISHER/string()"/>
            </xsl:map>
          </xsl:variable>
          <books authors="{$m('authors')}" titles="{$m('titles')}" prices="{$m('prices')}" publishers="{$m('publishers')}"/>     
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:map with computed key expressions -->
  
  <xsl:template name="m-003" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM">
          <xsl:variable name="m" as="map(*)">
            <xsl:map>
              <xsl:for-each select="*">
                <xsl:map-entry key="lower-case(local-name())" select="string(.)"/>
			        </xsl:for-each> 
            </xsl:map>
          </xsl:variable>
          <book author="{$m('author')}" title="{$m('title')}" price="{$m('price')}" publisher="{$m('publisher')}"/>     
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:map to compute max and min simultaneously -->
  
  <xsl:template name="m-004" use-when="$RUN" expand-text="yes">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
          <xsl:variable name="m" as="map(*)">
            <xsl:map>
              <xsl:map-entry key="'min'" select="min(outermost(//PRICE)/number())"/>
              <xsl:map-entry key="'max'" select="max(outermost(//PRICE)/number())"/>
            </xsl:map>
          </xsl:variable>
          <min>{$m('min')}</min> 
          <max>{$m('max')}</max>    
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:source-document, use xsl:map selecting an empty sequence -->
  
  <xsl:template name="m-005" use-when="$RUN" expand-text="yes">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
          <xsl:variable name="m" as="map(*)">
            <xsl:map>
              <xsl:map-entry key="'min'" select="outermost(//PRICEZZZ)/number()"/>
              <xsl:map-entry key="'max'" select="outermost(//PRICEZZZ)/number()"/>
            </xsl:map>
          </xsl:variable>
          <min>{count($m('min'))}</min> 
          <max>{count($m('max'))}</max>    
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:map with consuming key expressions -->
  
  <xsl:template name="m-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM[1]">
          <xsl:variable name="m" as="map(*)">
            <xsl:map>
              <xsl:map-entry key="AUTHOR" select="true()"/>
              <xsl:map-entry key="PUBLISHER" select="true()"/>
              <xsl:map-entry key="TITLE" select="true()"/>
            </xsl:map>
          </xsl:variable>
          <xsl:if test="$m('Jane Austen') and $m('Pride and Prejudice')"><ok/></xsl:if>    
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- Streaming xsl:map, error due to duplicate keys -->
  
  <xsl:template name="m-007" use-when="$RUN">
    <xsl:param name="a" select="'author'"/>
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM">
          <xsl:variable name="m" as="map(*)">
            <xsl:map>
              <xsl:map-entry key="'author'" select="string(AUTHOR)"/>
              <xsl:map-entry key="'title'" select="string(TITLE)"/>
              <xsl:map-entry key="'price'" select="number(PRICE)"/>
              <xsl:map-entry key="$a" select="string(PUBLISHER)"/>
            </xsl:map>
          </xsl:variable>
          <book author="{$m('author')}" title="{$m('title')}" price="{$m('price')}" publisher="{$m('publisher')}"/>     
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- Streaming map that contains another streaming map -->
  
  <xsl:template name="m-008" use-when="$RUN">
    <xsl:param name="a" as="item()">
      <xsl:map-entry key="'outcome'" select="'success'"/>
    </xsl:param>
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="BOOKLIST/BOOKS/ITEM">
          <xsl:variable name="m" as="map(*)">
            <xsl:map>
              <xsl:apply-templates mode="m-008" select="."/>
              <xsl:sequence select="$a"/>
            </xsl:map>
          </xsl:variable>
          <book author="{$m('AUTHOR')}" title="{$m('TITLE')}" price="{$m('PRICE')}" publisher="{$m('PUBLISHER')}" outcome="{$m('outcome')}"/>     
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="ITEM" mode="m-008">
    <xsl:map>
      <xsl:for-each select="*">
        <xsl:map-entry key="name(.)" select="data(.)"/>
      </xsl:for-each>
    </xsl:map>
  </xsl:template>
  
  <xsl:mode name="m-008" streamable="yes"/>
  
  <!-- Streaming map, type error because the input sequence includes a non-map -->
  
  <xsl:template name="m-009">
    <xsl:call-template name="m-008">
      <xsl:with-param name="a" select="93.7"/>
    </xsl:call-template>
  </xsl:template>
       
  

  
</xsl:transform>  